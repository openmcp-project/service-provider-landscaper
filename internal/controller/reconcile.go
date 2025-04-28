package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/cluster"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"
)

const (
	landscaperDomain    = "landscaper.services.openmcp.cloud"
	landscaperFinalizer = landscaperDomain + "/finalizer"
	landscaperOperation = landscaperDomain + "/operation"
	operationReconcile  = "reconcile"
)

func (r *LandscaperReconciler) reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := logging.FromContextOrPanic(ctx)

	ls := &v1alpha1.Landscaper{}
	if err := r.OnboardingClient.Get(ctx, req.NamespacedName, ls); err != nil {
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
	return res, err
}

func (r *LandscaperReconciler) handleCreateUpdateOperation(ctx context.Context, ls *v1alpha1.Landscaper) (reconcile.Result, error) {
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

	initConditions(ls)

	mcpClusterAccess, err := cluster.MCPCluster(ctx, client.ObjectKeyFromObject(ls), r.OnboardingClient)
	if err != nil {
		return reconcile.Result{}, err
	}

	conf, err := r.createConfig(ls, r.WorkloadCluster, mcpClusterAccess)
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
		return reconcile.Result{}, err
	}

	mcpCluster, err := cluster.MCPCluster(ctx, client.ObjectKeyFromObject(ls), r.OnboardingClient)
	if err != nil {
		return reconcile.Result{}, err
	}

	conf, err := r.createConfig(ls, r.WorkloadCluster, mcpCluster)
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

	// TODO: remove workload cluster request

	if err := r.removeFinalizer(ctx, ls); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *LandscaperReconciler) updateStatus(ctx context.Context, ls *v1alpha1.Landscaper) error {
	log := logging.FromContextOrPanic(ctx)
	if err := r.OnboardingClient.Status().Update(ctx, ls); err != nil {
		log.Error(err, "failed to update status of landscaper resource")
		return fmt.Errorf("failed to update status of landscaper resource %s/%s: %w", ls.Namespace, ls.Name, err)
	}
	return nil
}

func (r *LandscaperReconciler) ensureFinalizer(ctx context.Context, ls *v1alpha1.Landscaper) error {
	if !controllerutil.ContainsFinalizer(ls, landscaperFinalizer) && ls.DeletionTimestamp.IsZero() {
		controllerutil.AddFinalizer(ls, landscaperFinalizer)
		if err := r.OnboardingClient.Update(ctx, ls); err != nil {
			log := logging.FromContextOrPanic(ctx)
			log.Error(err, "failed to add finalizer to landscaper resource")
			return fmt.Errorf("failed to add finalizer to landscaper resource %s/%s: %w", ls.Namespace, ls.Name, err)
		}
	}
	return nil
}

func (r *LandscaperReconciler) removeFinalizer(ctx context.Context, ls *v1alpha1.Landscaper) error {
	if controllerutil.ContainsFinalizer(ls, landscaperFinalizer) {
		controllerutil.RemoveFinalizer(ls, landscaperFinalizer)
		if err := r.OnboardingClient.Update(ctx, ls); err != nil {
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
		if err := r.OnboardingClient.Status().Update(ctx, ls); err != nil {
			log := logging.FromContextOrPanic(ctx)
			log.Error(err, "failed to update observed generation for landscaper resource")
			return fmt.Errorf("failed to update observed generation for landscaper resource %s/%s: %w", ls.Namespace, ls.Name, err)
		}
	}
	return nil
}

func (r *LandscaperReconciler) checkReconcileAnnotation(ctx context.Context, ls *v1alpha1.Landscaper) error {
	if controller.HasAnnotationWithValue(ls, landscaperOperation, operationReconcile) && ls.Status.Phase == v1alpha1.Ready {
		log := logging.FromContextOrPanic(ctx)
		ls.Status.Phase = v1alpha1.Progressing
		if err := r.updateStatus(ctx, ls); err != nil {
			log.Error(err, "failed to handle reconcile annotation: unable to change phase of landscaper resource to progressing")
			return fmt.Errorf("failed to handle reconcile annotation: unable to change phase of landscaper resource %s/%s to progressing: %w", ls.Namespace, ls.Name, err)
		}

		if err := controller.EnsureAnnotation(ctx, r.OnboardingClient, ls, landscaperOperation, operationReconcile, true, controller.DELETE); err != nil {
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
		if err := r.OnboardingClient.Update(ctx, ls); err != nil {
			return fmt.Errorf("failed to set instance idfor landscaper resource %s/%s: %w", ls.Namespace, ls.Name, err)
		}
	}
	return nil
}

func (r *LandscaperReconciler) createConfig(ls *v1alpha1.Landscaper, workloadCluster, mcpCluster cluster.Cluster) (*instance.Configuration, error) {
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
		HostClusterDomain: r.WorkloadClusterDomain,
		Landscaper: instance.LandscaperConfig{
			Controller: instance.ControllerConfig{
				Image: v1alpha1.ImageConfiguration{
					Image: r.LandscaperProviderConfig.LandscaperController.Image,
				},
				Resources:     resources,
				ResourcesMain: resources,
			},
			WebhooksServer: instance.WebhooksServerConfig{
				Image: v1alpha1.ImageConfiguration{
					Image: r.LandscaperProviderConfig.LandscaperWebhooksServer.Image,
				},
				Resources: resources,
			},
		},
		ManifestDeployer: instance.ManifestDeployerConfig{
			Image: v1alpha1.ImageConfiguration{
				Image: r.LandscaperProviderConfig.ManifestDeployer.Image,
			},
			Resources: resources,
		},
		HelmDeployer: instance.HelmDeployerConfig{
			Image: v1alpha1.ImageConfiguration{
				Image: r.LandscaperProviderConfig.HelmDeployer.Image,
			},
			Resources: resources,
		},
	}
	return conf, nil
}
