package controller

import (
	"context"

	"github.com/openmcp-project/controller-utils/pkg/logging"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/cluster"
)

const (
	controllerName = "LandscaperProvider"
)

// LandscaperReconciler reconciles a Landscaper object
type LandscaperReconciler struct {
	OnboardingClient      client.Client
	WorkloadCluster       cluster.Cluster
	WorkloadClusterDomain string

	Scheme                   *runtime.Scheme
	LandscaperProviderConfig *api.LandscaperProviderConfiguration
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
		Named(controllerName).
		Complete(r)
}
