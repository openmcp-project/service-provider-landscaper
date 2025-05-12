package controller

import (
	"context"

	"github.com/openmcp-project/controller-utils/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"

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

// +kubebuilder:rbac:groups=landscaper.services.openmcp.cloud,resources=landscapers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscaper.services.openmcp.cloud,resources=landscapers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscaper.services.openmcp.cloud,resources=landscapers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Landscaper object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *LandscaperReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logging.FromContextOrPanic(ctx).WithName(controllerName)
	ctx = logging.NewContext(ctx, log)
	log.Debug("Starting reconcile")

	return r.reconcile(ctx, req)
}

// SetupWithManager sets up the controller with the Manager.
func (r *LandscaperReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Landscaper{}).
		WatchesRawSource(source.Kind(r.PlatformCluster.Cluster().GetCache(), &api.ProviderConfig{},
			handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, providerConfig *api.ProviderConfig) []ctrl.Request {
				// Find all Landscaper resources referencing this ProviderConfig
				var requests []reconcile.Request
				landscapers := &api.LandscaperList{}
				if err := r.OnboardingCluster.Client().List(ctx, landscapers); err != nil {
					return nil
				}

				for _, landscaper := range landscapers.Items {
					if landscaper.Status.ProviderConfigRef != nil && landscaper.Status.ProviderConfigRef.Name == providerConfig.Name {
						requests = append(requests, reconcile.Request{
							NamespacedName: client.ObjectKey{
								Namespace: landscaper.Namespace,
								Name:      landscaper.Name,
							},
						})
					}
				}
				return requests
			}), controller.ToTypedPredicate[*api.ProviderConfig](predicate.GenerationChangedPredicate{}),
		)).
		Named(controllerName).
		Complete(r)
}
