package controller

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/openmcp-project/service-provider-landscaper/internal/dns"

	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	rbac "k8s.io/api/rbac/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/openmcp-project/service-provider-landscaper/api/install"

	"github.com/openmcp-project/controller-utils/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/openmcp-project/service-provider-landscaper/api/v1alpha2"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/openmcp-project/controller-utils/pkg/logging"

	"github.com/openmcp-project/openmcp-operator/lib/clusteraccess"
)

const (
	controllerName = "LandscaperProvider"
)

// LandscaperReconciler reconciles a Landscaper object
type LandscaperReconciler struct {
	PlatformCluster         *clusters.Cluster
	OnboardingCluster       *clusters.Cluster
	ClusterAccessReconciler clusteraccess.Reconciler
	Scheme                  *runtime.Scheme
	DNSReconciler           *dns.Reconciler
	ProviderName            string
	ProviderNamespace       string

	InstanceClusterAccess InstanceClusterAccess
}

// The InstanceClusterAccess interface provides access to the MCP and Workload clusters for the Landscaper provider.
// This indirection is needed for injecting fake clusters in tests.
type InstanceClusterAccess interface {
	MCPCluster(ctx context.Context, req reconcile.Request) (*clusters.Cluster, error)
	WorkloadCluster(ctx context.Context, req reconcile.Request) (*clusters.Cluster, error)
}

type defaultInstanceClusterAccess struct {
	clusterAccessReconciler clusteraccess.Reconciler
}

func (d *defaultInstanceClusterAccess) MCPCluster(ctx context.Context, req reconcile.Request) (*clusters.Cluster, error) {
	return d.clusterAccessReconciler.MCPCluster(ctx, req)
}

func (d *defaultInstanceClusterAccess) WorkloadCluster(ctx context.Context, req reconcile.Request) (*clusters.Cluster, error) {
	return d.clusterAccessReconciler.WorkloadCluster(ctx, req)
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
	mcpScheme := runtime.NewScheme()
	install.InstallProviderAPIs(mcpScheme)
	utilruntime.Must(clientgoscheme.AddToScheme(mcpScheme))

	workloadScheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(workloadScheme))
	utilruntime.Must(gatewayv1.Install(workloadScheme))
	utilruntime.Must(gatewayv1alpha2.Install(workloadScheme))

	r.ClusterAccessReconciler = clusteraccess.NewClusterAccessReconciler(r.PlatformCluster.Client(), v1alpha2.LandscaperProviderName)
	r.ClusterAccessReconciler.
		WithMCPScheme(mcpScheme).
		WithWorkloadScheme(workloadScheme).
		WithRetryInterval(10 * time.Second).
		WithMCPPermissions(getMCPPermissions()).
		WithWorkloadPermissions(getWorkloadPermissions())

	r.InstanceClusterAccess = &defaultInstanceClusterAccess{
		clusterAccessReconciler: r.ClusterAccessReconciler,
	}

	r.DNSReconciler = dns.NewReconciler()

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.Landscaper{}).
		WatchesRawSource(source.Kind(r.PlatformCluster.Cluster().GetCache(), &v1alpha2.ProviderConfig{},
			handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, providerConfig *v1alpha2.ProviderConfig) []ctrl.Request {
				log := logging.Wrap(mgr.GetLogger()).WithName(controllerName + "/ProviderConfig")

				if log.Enabled(logging.DEBUG) {
					providerConfigType, hasLabel := controller.GetLabel(providerConfig, v1alpha2.ProviderConfigTypeLabel)
					isDefault := hasLabel && providerConfigType == v1alpha2.DefaultProviderConfigValue
					log.Debug("Starting reconcile", "providerConfig", providerConfig.Name, "isDefault", isDefault)
				}

				// Find all Landscaper resources referencing this ProviderConfig
				landscapers := &v1alpha2.LandscaperList{}
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
							v1alpha2.LandscaperOperation, v1alpha2.OperationReconcile,
							true, controller.OVERWRITE); err != nil {
							log.Error(err, "Failed to set reconcile annotation for Landscaper resource", "landscaper", landscaper.Name, "namespace", landscaper.Namespace)
							return nil
						}

						// don't add the request since it will already be reconciled by setting the annotation
					}
				}
				return nil
			}), controller.ToTypedPredicate[*v1alpha2.ProviderConfig](predicate.GenerationChangedPredicate{}),
		)).
		Named(controllerName).
		Complete(r)
}

func getMCPPermissions() []clustersv1alpha1.PermissionsRequest {
	defaultVerbs := []string{"get", "list", "watch", "create", "update", "patch", "delete"}

	return []clustersv1alpha1.PermissionsRequest{
		{
			Rules: []rbac.PolicyRule{
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs:     defaultVerbs,
				},
				{
					APIGroups: []string{"landscaper.gardener.cloud"},
					Resources: []string{"*"},
					Verbs:     defaultVerbs,
				},
				{
					APIGroups: []string{""},
					Resources: []string{"secrets", "configmaps"},
					Verbs:     defaultVerbs,
				},
				{
					APIGroups: []string{""},
					Resources: []string{"serviceaccounts"},
					Verbs:     defaultVerbs,
				},
				{
					APIGroups: []string{""},
					Resources: []string{"serviceaccounts/token"},
					Verbs:     []string{"create"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     defaultVerbs,
				},
				{
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{"clusterroles", "clusterrolebindings"},
					Verbs:     defaultVerbs,
				},
				{
					APIGroups: []string{""},
					Resources: []string{"events"},
					Verbs:     defaultVerbs,
				},
				{
					APIGroups: []string{"admissionregistration.k8s.io"},
					Resources: []string{"validatingwebhookconfigurations"},
					Verbs:     defaultVerbs,
				},
			},
		},
	}
}

func getWorkloadPermissions() []clustersv1alpha1.PermissionsRequest {
	return []clustersv1alpha1.PermissionsRequest{
		{
			// TODO: refine the permissions for the Workload cluster
			Rules: []rbac.PolicyRule{
				{
					APIGroups: []string{"*"},
					Resources: []string{"*"},
					Verbs:     []string{"*"},
				},
			},
		},
	}
}
