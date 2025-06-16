package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/openmcp-project/controller-utils/pkg/clusters"

	"github.com/openmcp-project/controller-utils/pkg/controller"
	"github.com/openmcp-project/controller-utils/pkg/logging"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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

	oldStatus := ls.Status.DeepCopy()
	var status *reconcileStatus

	if ls.DeletionTimestamp.IsZero() {
		res, status, err = r.handleCreateUpdateOperation(ctx, ls)
	} else {
		res, status, err = r.handleDeleteOperation(ctx, ls)
	}

	if status != nil {
		status.convertToLandscaperStatus(&ls.Status)
	}

	updateErr := r.updateStatus(ctx, ls, oldStatus)
	err = errors.Join(err, updateErr)
	if err != nil {
		return reconcile.Result{}, err
	} else {
		return res, nil
	}
}

func (r *LandscaperReconciler) handleCreateUpdateOperation(ctx context.Context,
	ls *v1alpha1.Landscaper) (reconcile.Result, *reconcileStatus, error) {
	log := logging.FromContextOrPanic(ctx)

	status := newCreateOrUpdateStatus(ls.GetGeneration())

	if err := r.ensureFinalizer(ctx, ls); err != nil {
		return reconcile.Result{}, status, err
	}

	if err := r.checkReconcileAnnotation(ctx, ls); err != nil {
		return reconcile.Result{}, status, err
	}

	if err := r.ensureInstanceID(ctx, ls); err != nil {
		return reconcile.Result{}, status, err
	}

	providerConfig, err := r.getProviderConfigForLandscaper(ctx, ls, r.PlatformCluster)
	if err != nil {
		log.Error(err, "failed to get provider config for landscaper instance")
		status.setInstallProviderConfigError(err)
		return reconcile.Result{}, status, err
	}

	req := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(ls)}
	res, err := r.ClusterAccessReconciler.Reconcile(ctx, req)
	if err != nil {
		log.Error(err, "failed to reconcile cluster access for landscaper instance")
		status.setInstallClusterAccessError(err)
		return reconcile.Result{}, status, err
	}

	if res.RequeueAfter > 0 {
		status.setInstallWaitForClusterAccessReady()
		return res, status, nil
	}

	mcpCluster, err := r.InstanceClusterAccess.MCPCluster(ctx, req)
	if err != nil {
		log.Error(err, "failed to get MCP cluster for landscaper instance")
		status.setInstallClusterAccessError(err)
		return reconcile.Result{}, status, err
	}

	workloadCluster, err := r.InstanceClusterAccess.WorkloadCluster(ctx, req)
	if err != nil {
		log.Error(err, "failed to get Workload cluster for landscaper instance")
		status.setInstallClusterAccessError(err)
		return reconcile.Result{}, status, err
	}

	conf, err := r.createConfig(ls, mcpCluster, workloadCluster, providerConfig)
	if err != nil {
		log.Error(err, "failed to create configuration for landscaper instance")
		status.setInstallConfigurationError(err)
		return reconcile.Result{}, status, err
	}

	if err := instance.InstallLandscaperInstance(ctx, conf); err != nil {
		log.Error(err, "failed to install landscaper instance")
		status.setInstallFailed(err)
		return ctrl.Result{}, status, err
	}
	log.Debug("landscaper instance has been installed")
	status.setInstalled()

	if readinessCheckResult := instance.CheckReadiness(ctx, conf); !readinessCheckResult.IsReady() {
		log.Debug("landscaper instance is not yet ready")
		status.setWaitForReadinessCheck(readinessCheckResult)
		return ctrl.Result{RequeueAfter: 40 * time.Second}, status, nil
	}

	ls.Status.Phase = v1alpha1.PhaseReady
	log.Debug("landscaper instance has become ready")
	status.setReady()

	return reconcile.Result{}, status, nil
}

func (r *LandscaperReconciler) handleDeleteOperation(ctx context.Context, ls *v1alpha1.Landscaper) (reconcile.Result, *reconcileStatus, error) {
	log := logging.FromContextOrPanic(ctx)

	status := newDeleteStatus(ls.GetGeneration())

	req := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(ls)}
	res, err := r.ClusterAccessReconciler.Reconcile(ctx, req)
	if err != nil {
		log.Error(err, "failed to reconcile cluster access for landscaper instance deletion")
		status.setUninstallClusterAccessError(err)
		return res, status, err
	}

	if res.RequeueAfter > 0 {
		status.setUninstallWaitForClusterAccessReady()
		return res, status, nil
	}

	mcpCluster, err := r.ClusterAccessReconciler.MCPCluster(ctx, req)
	if err != nil {
		log.Error(err, "failed to get MCP cluster for landscaper instance")
		status.setUninstallClusterAccessError(err)
		return reconcile.Result{}, status, err
	}

	workloadCluster, err := r.ClusterAccessReconciler.WorkloadCluster(ctx, req)
	if err != nil {
		log.Error(err, "failed to get Workload cluster for landscaper instance")
		status.setUninstallClusterAccessError(err)
		return reconcile.Result{}, status, err
	}

	providerConfig, err := r.getProviderConfigForLandscaper(ctx, ls, r.PlatformCluster)
	if err != nil {
		log.Error(err, "failed to get provider config for landscaper instance")
		status.setUninstallProviderConfigError(err)
		return reconcile.Result{}, status, err
	}

	conf, err := r.createConfig(ls, mcpCluster, workloadCluster, providerConfig)
	if err != nil {
		log.Error(err, "failed to create configuration to uninstall landscaper instance")
		status.setUninstallConfigurationError(err)
		return reconcile.Result{}, status, err
	}

	if err = instance.UninstallLandscaperInstance(ctx, conf); err != nil {
		log.Error(err, "failed to uninstall landscaper instance")
		status.setUninstallFailed(err)
		return reconcile.Result{}, status, err
	}
	log.Debug("landscaper instance has been uninstalled")
	status.setUninstalled()

	result, err := r.ClusterAccessReconciler.ReconcileDelete(ctx, req)
	if err != nil {
		log.Error(err, "failed to reconcile cluster access for landscaper instance deletion")
		status.setUninstallClusterAccessError(err)
		return result, status, err
	}

	if result.RequeueAfter > 0 {
		return result, status, nil
	}

	if err = r.removeFinalizer(ctx, ls); err != nil {
		return reconcile.Result{}, status, err
	}

	return reconcile.Result{}, status, nil
}

func (r *LandscaperReconciler) updateStatus(ctx context.Context, ls *v1alpha1.Landscaper, oldStatus *v1alpha1.LandscaperStatus) error {
	log := logging.FromContextOrPanic(ctx)
	if !reflect.DeepEqual(oldStatus, &ls.Status) {
		if err := r.OnboardingCluster.Client().Status().Update(ctx, ls); err != nil {
			log.Error(err, "failed to update status of landscaper resource")
			return fmt.Errorf("failed to update status of landscaper resource %s/%s: %w", ls.Namespace, ls.Name, err)
		}
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

func (r *LandscaperReconciler) checkReconcileAnnotation(ctx context.Context, ls *v1alpha1.Landscaper) error {
	if controller.HasAnnotationWithValue(ls, v1alpha1.LandscaperOperation, v1alpha1.OperationReconcile) {
		log := logging.FromContextOrPanic(ctx)

		if err := controller.EnsureAnnotation(ctx, r.OnboardingCluster.Client(), ls, v1alpha1.LandscaperOperation, v1alpha1.OperationReconcile, true, controller.DELETE); err != nil {
			log.Error(err, "failed to remove reconcile annotation from landscaper resource")
			return fmt.Errorf("failed to remove reconcile annotation from landscaper resource %s/%s: %w", ls.Namespace, ls.Name, err)
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

	providerConfig := &v1alpha1.ProviderConfig{}
	if err := platformCluster.Client().Get(ctx, client.ObjectKey{Name: providerConfigName}, providerConfig); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("provider config %s not found", providerConfigName)
		}
		return nil, fmt.Errorf("failed to get provider config %s: %w", providerConfigName, err)
	}

	oldStatus := ls.Status.DeepCopy()

	ls.Status.ProviderConfigRef = &core.LocalObjectReference{
		Name: providerConfigName,
	}

	ls.Status.LandscaperComponents = []v1alpha1.LandscaperComponent{
		{
			Name:    v1alpha1.GetControllerName(),
			Version: providerConfig.GetControllerVersion(),
		},
		{
			Name:    v1alpha1.GetWebhooksServerName(),
			Version: providerConfig.GetWebhooksServerVersion(),
		},
		{
			Name:    v1alpha1.GetManifestDeployerName(),
			Version: providerConfig.GetManifestDeployerVersion(),
		},
		{
			Name:    v1alpha1.GetHelmDeployerName(),
			Version: providerConfig.GetHelmDeployerVersion(),
		},
	}

	if err := r.updateStatus(ctx, ls, oldStatus); err != nil {
		return nil, err
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
