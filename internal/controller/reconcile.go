package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmcp-project/controller-utils/pkg/clusters"

	"github.com/openmcp-project/controller-utils/pkg/controller"
	"github.com/openmcp-project/controller-utils/pkg/logging"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
	"github.com/openmcp-project/service-provider-landscaper/internal/installer/instance"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"
)

func (r *LandscaperReconciler) reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := logging.FromContextOrPanic(ctx)

	ls := &v1alpha1.Landscaper{}
	if err := r.OnboardingCluster.Client().Get(ctx, req.NamespacedName, ls); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		log.Error(err, "failed to get landscaper resource")
		return reconcile.Result{}, err
	}

	if ls.DeletionTimestamp.IsZero() {
		res, err = r.handleCreateUpdateOperation(ctx, ls)
	} else {
		res, err = r.handleDeleteOperation(ctx, ls)
	}

	updateErr := r.updateStatus(ctx, ls)
	err = errors.Join(err, updateErr)
	if err != nil {
		return reconcile.Result{}, err
	} else {
		return res, nil
	}
}

func (r *LandscaperReconciler) handleCreateUpdateOperation(ctx context.Context,
	ls *v1alpha1.Landscaper) (reconcile.Result, error) {
	log := logging.FromContextOrPanic(ctx)

	if err := r.ensureFinalizer(ctx, ls); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.checkReconcileAnnotation(ctx, ls); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.observeGeneration(ctx, ls); err != nil {
		return reconcile.Result{}, err
	}

	if ls.Status.Phase == v1alpha1.Ready {
		return reconcile.Result{}, nil
	}

	if err := r.ensureInstanceID(ctx, ls); err != nil {
		return reconcile.Result{}, err
	}

	req := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(ls)}
	res, err := r.ClusterAccessReconciler.Reconcile(ctx, req)
	if err != nil {
		log.Error(err, "failed to reconcile cluster access for landscaper instance")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Installed",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "ClusterAccessError",
			Message:            err.Error(),
		})
		return reconcile.Result{}, err
	}

	if res.RequeueAfter > 0 {
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Installed",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "WaitingForClusterAccessToBeReady",
			Message:            "MCP and/or Workload Clusters are not yet ready",
		})
		return res, nil
	}

	mcpCluster, err := r.ClusterAccessReconciler.MCPCluster(ctx, req)
	if err != nil {
		log.Error(err, "failed to get MCP cluster for landscaper instance")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Installed",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "MCPClusterError",
			Message:            err.Error(),
		})
		return reconcile.Result{}, err
	}

	workloadCluster, err := r.ClusterAccessReconciler.WorkloadCluster(ctx, req)
	if err != nil {
		log.Error(err, "failed to get Workload cluster for landscaper instance")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Installed",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "WorkloadClusterError",
			Message:            err.Error(),
		})
		return reconcile.Result{}, err
	}

	initConditions(ls)

	providerConfig, err := r.getProviderConfigForLandscaper(ctx, ls, r.PlatformCluster)
	if err != nil {
		log.Error(err, "failed to get provider config for landscaper instance")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Installed",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "ProviderConfigError",
			Message:            err.Error(),
		})
		return reconcile.Result{}, err
	}

	conf, err := r.createConfig(ls, mcpCluster, workloadCluster, providerConfig)
	if err != nil {
		log.Error(err, "failed to create configuration for landscaper instance")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Installed",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "ConfigurationError",
			Message:            err.Error(),
		})
		return reconcile.Result{}, err
	}

	if err := instance.InstallLandscaperInstance(ctx, conf); err != nil {
		log.Error(err, "failed to install landscaper instance")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Installed",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "InstallFailed",
			Message:            err.Error(),
		})
		return ctrl.Result{}, err
	}
	log.Debug("landscaper instance has been installed")
	apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
		Type:               "Installed",
		Status:             meta.ConditionTrue,
		ObservedGeneration: ls.Generation,
		Reason:             "LandscaperInstalled",
		Message:            "Landscaper has been installed",
	})

	if readinessCheckResult := instance.CheckReadiness(ctx, conf); !readinessCheckResult.IsReady() {
		log.Debug("landscaper instance is not yet ready")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Ready",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "WaitingForLandscaperToBecomeReady",
			Message:            readinessCheckResult.Message(),
		})
		return ctrl.Result{RequeueAfter: 40 * time.Second}, nil
	}

	ls.Status.Phase = v1alpha1.Ready
	log.Debug("landscaper instance has become ready")
	apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
		Type:               "Ready",
		Status:             meta.ConditionTrue,
		ObservedGeneration: ls.Generation,
		Reason:             "LandscaperReady",
		Message:            "Landscaper is ready",
	})

	return reconcile.Result{}, nil
}

func (r *LandscaperReconciler) handleDeleteOperation(ctx context.Context, ls *v1alpha1.Landscaper) (reconcile.Result, error) {
	log := logging.FromContextOrPanic(ctx)

	if err := r.ensurePhaseTerminating(ctx, ls); err != nil {
		log.Error(err, "failed to ensure landscaper instance is in terminating phase")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Uninstalled",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "PhaseError",
			Message:            err.Error(),
		})
		return reconcile.Result{}, err
	}

	req := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(ls)}
	res, err := r.ClusterAccessReconciler.Reconcile(ctx, req)
	if err != nil {
		log.Error(err, "failed to reconcile cluster access for landscaper instance deletion")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Uninstalled",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "ClusterAccessError",
			Message:            err.Error(),
		})
		return res, err
	}

	if res.RequeueAfter > 0 {
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Uninstalled",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "WaitingForClusterAccessToBeReady",
			Message:            "MCP and/or Workload Clusters are not yet ready",
		})
		return res, nil
	}

	mcpCluster, err := r.ClusterAccessReconciler.MCPCluster(ctx, req)
	if err != nil {
		log.Error(err, "failed to get MCP cluster for landscaper instance")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Uninstalled",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "MCPClusterError",
			Message:            err.Error(),
		})
		return reconcile.Result{}, err
	}

	workloadCluster, err := r.ClusterAccessReconciler.WorkloadCluster(ctx, req)
	if err != nil {
		log.Error(err, "failed to get Workload cluster for landscaper instance")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Uninstalled",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "WorkloadClusterError",
			Message:            err.Error(),
		})
		return reconcile.Result{}, err
	}

	providerConfig, err := r.getProviderConfigForLandscaper(ctx, ls, r.PlatformCluster)
	if err != nil {
		log.Error(err, "failed to get provider config for landscaper instance")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Uninstalled",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "ProviderConfigError",
			Message:            err.Error(),
		})
		return reconcile.Result{}, err
	}

	conf, err := r.createConfig(ls, mcpCluster, workloadCluster, providerConfig)
	if err != nil {
		log.Error(err, "failed to create configuration to uninstall landscaper instance")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Uninstalled",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "ConfigurationError",
			Message:            err.Error(),
		})
		return reconcile.Result{}, err
	}

	if err := instance.UninstallLandscaperInstance(ctx, conf); err != nil {
		log.Error(err, "failed to uninstall landscaper instance")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Uninstalled",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "UninstallFailed",
			Message:            err.Error(),
		})
		return reconcile.Result{}, err
	}
	log.Debug("landscaper instance has been uninstalled")
	apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
		Type:               "Uninstalled",
		Status:             meta.ConditionTrue,
		ObservedGeneration: ls.Generation,
		Reason:             "LandscaperUninstalled",
		Message:            "Landscaper has been uninstalled",
	})

	result, err := r.ClusterAccessReconciler.ReconcileDelete(ctx, req)
	if err != nil {
		log.Error(err, "failed to reconcile cluster access for landscaper instance deletion")
		apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
			Type:               "Uninstalled",
			Status:             meta.ConditionFalse,
			ObservedGeneration: ls.Generation,
			Reason:             "ClusterAccessError",
			Message:            err.Error(),
		})
		return result, err
	}

	if result.RequeueAfter > 0 {
		return result, nil
	}

	if err := r.removeFinalizer(ctx, ls); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *LandscaperReconciler) updateStatus(ctx context.Context, ls *v1alpha1.Landscaper) error {
	log := logging.FromContextOrPanic(ctx)
	if err := r.OnboardingCluster.Client().Status().Update(ctx, ls); err != nil {
		log.Error(err, "failed to update status of landscaper resource")
		return fmt.Errorf("failed to update status of landscaper resource %s/%s: %w", ls.Namespace, ls.Name, err)
	}
	return nil
}

func (r *LandscaperReconciler) ensureFinalizer(ctx context.Context, ls *v1alpha1.Landscaper) error {
	if !controllerutil.ContainsFinalizer(ls, v1alpha1.LandscaperFinalizer) && ls.DeletionTimestamp.IsZero() {
		controllerutil.AddFinalizer(ls, v1alpha1.LandscaperFinalizer)
		if err := r.OnboardingCluster.Client().Update(ctx, ls); err != nil {
			log := logging.FromContextOrPanic(ctx)
			log.Error(err, "failed to add finalizer to landscaper resource")
			return fmt.Errorf("failed to add finalizer to landscaper resource %s/%s: %w", ls.Namespace, ls.Name, err)
		}
	}
	return nil
}

func (r *LandscaperReconciler) removeFinalizer(ctx context.Context, ls *v1alpha1.Landscaper) error {
	if controllerutil.ContainsFinalizer(ls, v1alpha1.LandscaperFinalizer) {
		controllerutil.RemoveFinalizer(ls, v1alpha1.LandscaperFinalizer)
		if err := r.OnboardingCluster.Client().Update(ctx, ls); err != nil {
			log := logging.FromContextOrPanic(ctx)
			log.Error(err, "failed to remove finalizer from landscaper resource")
			return fmt.Errorf("failed to remove finalizer from landscaper resource %s/%s: %w", ls.Namespace, ls.Name, err)
		}
	}
	return nil
}

func (r *LandscaperReconciler) observeGeneration(ctx context.Context, ls *v1alpha1.Landscaper) error {
	if ls.Status.ObservedGeneration != ls.Generation {
		ls.Status.ObservedGeneration = ls.Generation
		ls.Status.Phase = v1alpha1.Progressing
		if err := r.OnboardingCluster.Client().Status().Update(ctx, ls); err != nil {
			log := logging.FromContextOrPanic(ctx)
			log.Error(err, "failed to update observed generation for landscaper resource")
			return fmt.Errorf("failed to update observed generation for landscaper resource %s/%s: %w", ls.Namespace, ls.Name, err)
		}
	}
	return nil
}

func (r *LandscaperReconciler) checkReconcileAnnotation(ctx context.Context, ls *v1alpha1.Landscaper) error {
	if controller.HasAnnotationWithValue(ls, v1alpha1.LandscaperOperation, v1alpha1.OperationReconcile) && ls.Status.Phase == v1alpha1.Ready {
		log := logging.FromContextOrPanic(ctx)
		ls.Status.Phase = v1alpha1.Progressing
		if err := r.updateStatus(ctx, ls); err != nil {
			log.Error(err, "failed to handle reconcile annotation: unable to change phase of landscaper resource to progressing")
			return fmt.Errorf("failed to handle reconcile annotation: unable to change phase of landscaper resource %s/%s to progressing: %w", ls.Namespace, ls.Name, err)
		}

		if err := controller.EnsureAnnotation(ctx, r.OnboardingCluster.Client(), ls, v1alpha1.LandscaperOperation, v1alpha1.OperationReconcile, true, controller.DELETE); err != nil {
			log.Error(err, "failed to remove reconcile annotation from landscaper resource")
			return fmt.Errorf("failed to remove reconcile annotation from landscaper resource %s/%s: %w", ls.Namespace, ls.Name, err)
		}
	}
	return nil
}

func (r *LandscaperReconciler) ensurePhaseTerminating(ctx context.Context, ls *v1alpha1.Landscaper) error {
	if ls.Status.ObservedGeneration != ls.Generation || ls.Status.Phase != v1alpha1.Terminating {
		ls.Status.ObservedGeneration = ls.Generation
		ls.Status.Phase = v1alpha1.Terminating
		if err := r.updateStatus(ctx, ls); err != nil {
			return err
		}
	}
	return nil
}

func (r *LandscaperReconciler) ensureInstanceID(ctx context.Context, ls *v1alpha1.Landscaper) error {
	if len(identity.GetInstanceID(ls)) == 0 {
		identity.SetInstanceID(ls, identity.ComputeInstanceID(ls))
		if err := r.OnboardingCluster.Client().Update(ctx, ls); err != nil {
			return fmt.Errorf("failed to set instance idfor landscaper resource %s/%s: %w", ls.Namespace, ls.Name, err)
		}
	}
	return nil
}

func (r *LandscaperReconciler) getProviderConfigForLandscaper(ctx context.Context, ls *v1alpha1.Landscaper, platformCluster *clusters.Cluster) (*v1alpha1.ProviderConfig, error) {
	var providerConfigName string

	// first, check if the landscaper already has a provider config reference in its status
	if ls.Status.ProviderConfigRef != nil {
		providerConfigName = ls.Status.ProviderConfigRef.Name
	}

	// check if the landscaper has a provider config reference in its spec, which shall override the one in the status
	if ls.Spec.ProviderConfigRef != nil {
		providerConfigName = ls.Spec.ProviderConfigRef.Name
	}

	// if provider config name is empty, find the one with label "landscaper.services.openmcp.cloud/type=default"
	if providerConfigName == "" {
		providerConfigList := &v1alpha1.ProviderConfigList{}
		if err := platformCluster.Client().List(ctx, providerConfigList, client.MatchingLabels{v1alpha1.ProviderConfigTypeLabel: v1alpha1.DefaultProviderConfigValue}); err != nil {
			return nil, fmt.Errorf("failed to list provider config resources: %w", err)
		}
		if len(providerConfigList.Items) == 0 {
			return nil, fmt.Errorf("no default provider config found")
		}
		providerConfigName = providerConfigList.Items[0].Name
	}

	// update the provider config reference in the landscaper status if it is not already set and equal to the one in the spec
	if ls.Status.ProviderConfigRef == nil || ls.Status.ProviderConfigRef.Name != providerConfigName {
		ls.Status.ProviderConfigRef = &core.LocalObjectReference{
			Name: providerConfigName,
		}
		if err := r.OnboardingCluster.Client().Status().Update(ctx, ls); err != nil {
			return nil, fmt.Errorf("failed to update provider config reference in landscaper status: %w", err)
		}
	}

	providerConfig := &v1alpha1.ProviderConfig{}
	if err := platformCluster.Client().Get(ctx, client.ObjectKey{Namespace: ls.Namespace, Name: providerConfigName}, providerConfig); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("provider config %s not found", providerConfigName)
		}
		return nil, fmt.Errorf("failed to get provider config %s: %w", providerConfigName, err)
	}

	return providerConfig, nil
}

func (r *LandscaperReconciler) createConfig(ls *v1alpha1.Landscaper, mcpCluster, workloadCluster *clusters.Cluster, providerConfig *v1alpha1.ProviderConfig) (*instance.Configuration, error) {
	inst := identity.Instance(identity.GetInstanceID(ls))

	cpu, err := resource.ParseQuantity("10m")
	if err != nil {
		return nil, err
	}
	memory, err := resource.ParseQuantity("30Mi")
	if err != nil {
		return nil, err
	}
	resources := core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceCPU:    cpu,
			core.ResourceMemory: memory,
		},
	}
	conf := &instance.Configuration{
		Instance:          inst,
		Version:           "v0.127.0",
		ResourceCluster:   mcpCluster,
		HostCluster:       workloadCluster,
		HostClusterDomain: providerConfig.Spec.WorkloadClusterDomain,
		Landscaper: instance.LandscaperConfig{
			Controller: instance.ControllerConfig{
				Image: v1alpha1.ImageConfiguration{
					Image: providerConfig.Spec.Deployment.LandscaperController.Image,
				},
				Resources:     resources,
				ResourcesMain: resources,
			},
			WebhooksServer: instance.WebhooksServerConfig{
				Image: v1alpha1.ImageConfiguration{
					Image: providerConfig.Spec.Deployment.LandscaperWebhooksServer.Image,
				},
				Resources: resources,
			},
		},
		ManifestDeployer: instance.ManifestDeployerConfig{
			Image: v1alpha1.ImageConfiguration{
				Image: providerConfig.Spec.Deployment.ManifestDeployer.Image,
			},
			Resources: resources,
		},
		HelmDeployer: instance.HelmDeployerConfig{
			Image: v1alpha1.ImageConfiguration{
				Image: providerConfig.Spec.Deployment.HelmDeployer.Image,
			},
			Resources: resources,
		},
	}
	return conf, nil
}
