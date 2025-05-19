package controller

import (
	"context"

	"github.com/openmcp-project/controller-utils/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/openmcp-project/controller-utils/pkg/logging"
)

const (
	controllerName = "LandscaperProvider"
)

// LandscaperReconciler reconciles a Landscaper object
type LandscaperReconciler struct {
	PlatformCluster       *clusters.Cluster
	OnboardingCluster     *clusters.Cluster
	WorkloadCluster       *clusters.Cluster
	WorkloadClusterDomain string

	Scheme *runtime.Scheme
}

//nolint:lll
// +kubebuilder:rbac:groups=landscaper.services.openmcp.cloud,resources=landscapers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscaper.services.openmcp.cloud,resources=landscapers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscaper.services.openmcp.cloud,resources=landscapers/finalizers,verbs=update

func (r *LandscaperReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logging.FromContextOrPanic(ctx).WithName(controllerName)
	ctx = logging.NewContext(ctx, log)
	log.Debug("Starting reconcile")

	return r.reconcile(ctx, req)
}

// SetupWithManager sets up the controller with the Manager.
func (r *LandscaperReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Landscaper{}).
		WatchesRawSource(source.Kind(r.PlatformCluster.Cluster().GetCache(), &v1alpha1.ProviderConfig{},
			handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, providerConfig *v1alpha1.ProviderConfig) []ctrl.Request {
				log := logging.Wrap(mgr.GetLogger()).WithName(controllerName + "/ProviderConfig")

				if log.Enabled(logging.DEBUG) {
					providerConfigType, hasLabel := controller.GetLabel(providerConfig, v1alpha1.ProviderConfigTypeLabel)
					isDefault := hasLabel && providerConfigType == v1alpha1.DefaultProviderConfigValue
					log.Debug("Starting reconcile", "providerConfig", providerConfig.Name, "isDefault", isDefault)
				}

				// Find all Landscaper resources referencing this ProviderConfig
				landscapers := &v1alpha1.LandscaperList{}
				if err := r.OnboardingCluster.Client().List(ctx, landscapers); err != nil {
					log.Error(err, "Failed to list Landscaper resources")
					return nil
				}

				for _, landscaper := range landscapers.Items {
					if landscaper.Status.ProviderConfigRef != nil && landscaper.Status.ProviderConfigRef.Name == providerConfig.Name {
						// set the reconcile annotation for the landscaper
						log.Debug("Setting reconcile annotation for Landscaper resource", "landscaper", landscaper.Name, "namespace", landscaper.Namespace)

						if err := controller.EnsureAnnotation(
							ctx, r.OnboardingCluster.Client(),
							&landscaper,
							v1alpha1.LandscaperOperation, v1alpha1.OperationReconcile,
							true, controller.OVERWRITE); err != nil {
							log.Error(err, "Failed to set reconcile annotation for Landscaper resource", "landscaper", landscaper.Name, "namespace", landscaper.Namespace)
							return nil
						}

						// don't add the request since it will already be reconciled by setting the annotation
					}
				}
				return nil
			}), controller.ToTypedPredicate[*v1alpha1.ProviderConfig](predicate.GenerationChangedPredicate{}),
		)).
		Named(controllerName).
		Complete(r)
}
