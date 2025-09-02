package controller_test

import (
	"context"
	"time"

	libutils "github.com/openmcp-project/openmcp-operator/lib/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"

	testutils "github.com/openmcp-project/controller-utils/pkg/testing"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"github.com/openmcp-project/openmcp-operator/lib/clusteraccess"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/openmcp-project/controller-utils/pkg/clusters"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"

	lscontroller "github.com/openmcp-project/service-provider-landscaper/internal/controller"

	commonapi "github.com/openmcp-project/openmcp-operator/api/common"
)

const (
	controllerName = "test-controller"
)

func setDeploymentReady(ctx context.Context, deployment *appsv1.Deployment, c client.Client) {
	deployment.Status.Conditions = []appsv1.DeploymentCondition{
		{
			Type:   appsv1.DeploymentAvailable,
			Status: corev1.ConditionTrue,
		},
	}

	deployment.Status.Replicas = *deployment.Spec.Replicas
	deployment.Status.ReadyReplicas = *deployment.Spec.Replicas
	deployment.Status.AvailableReplicas = *deployment.Spec.Replicas
	deployment.Status.UpdatedReplicas = *deployment.Spec.Replicas
	deployment.Status.ObservedGeneration = deployment.Generation

	Expect(c.Status().Update(ctx, deployment)).To(Succeed())
}

type testInstanceClusterAccess struct {
	mcpCluster      *clusters.Cluster
	workloadCluster *clusters.Cluster
}

func (t *testInstanceClusterAccess) MCPCluster(ctx context.Context, req reconcile.Request) (*clusters.Cluster, error) {
	return t.mcpCluster, nil
}

func (t *testInstanceClusterAccess) WorkloadCluster(ctx context.Context, req reconcile.Request) (*clusters.Cluster, error) {
	return t.workloadCluster, nil
}

func buildTestEnvironmentReconcile(testdataDir string, objectsWithStatus ...client.Object) *testutils.Environment {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clustersv1alpha1.AddToScheme(scheme))
	utilruntime.Must(lsv1alpha1.AddToScheme(scheme))

	return testutils.NewEnvironmentBuilder().
		WithFakeClient(scheme).
		WithInitObjectPath("testdata", testdataDir).
		WithDynamicObjectsWithStatus(objectsWithStatus...).
		WithReconcilerConstructor(func(c client.Client) reconcile.Reconciler {
			permissions := []clustersv1alpha1.PermissionsRequest{
				{
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{"*"},
							Resources: []string{"*"},
							Verbs:     []string{"*"},
						},
					},
				},
			}

			car := clusteraccess.NewClusterAccessReconciler(c, controllerName)
			car.WithMCPScheme(scheme).
				WithWorkloadScheme(scheme).
				WithMCPPermissions(permissions).
				WithWorkloadPermissions(permissions).
				WithRetryInterval(1 * time.Second)

			platformCluster := clusters.NewTestClusterFromClient("platform", c)
			onboardingCluster := clusters.NewTestClusterFromClient("onboarding", c)

			r := &lscontroller.LandscaperReconciler{
				Scheme:                  scheme,
				ClusterAccessReconciler: car,
				PlatformCluster:         platformCluster,
				OnboardingCluster:       onboardingCluster,
				InstanceClusterAccess: &testInstanceClusterAccess{
					mcpCluster:      clusters.NewTestClusterFromClient("mcp", c),
					workloadCluster: clusters.NewTestClusterFromClient("workload", c),
				},
			}

			return r
		}).
		Build()
}

var _ = Describe("Landscaper Controller", func() {
	Context("CreateUpdate", func() {
		It("should set the finalizer and the provider config reference", func() {
			env := buildTestEnvironmentReconcile("test-01")

			req := reconcile.Request{
				NamespacedName: client.ObjectKey{
					Name:      "test",
					Namespace: "default",
				},
			}

			env.ShouldReconcile(req, "reconcile should not return an error and set finalizer")

			ls := &lsv1alpha1.Landscaper{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			}

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(ls), ls)).To(Succeed())
			Expect(ls.ObjectMeta.Finalizers).To(ContainElement(lsv1alpha1.LandscaperFinalizer))

			Expect(ls.Status.ProviderConfigRef.Name).To(Equal("default"))
			Expect(ls.Status.Phase).To(Equal(lsv1alpha1.PhaseProgressing))
			Expect(ls.Status.Conditions).To(HaveLen(2))
			Expect(ls.Status.Conditions[0].Type).To(Equal(lsv1alpha1.ConditionTypeInstalled))
			Expect(ls.Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(ls.Status.Conditions[1].Type).To(Equal(lsv1alpha1.ConditionTypeReady))
			Expect(ls.Status.Conditions[1].Status).To(Equal(metav1.ConditionUnknown))
		})

		It("should use an explicit provider config reference", func() {
			env := buildTestEnvironmentReconcile("test-02")

			req := reconcile.Request{
				NamespacedName: client.ObjectKey{
					Name:      "test",
					Namespace: "default",
				},
			}

			env.ShouldReconcile(req, "reconcile should not return an error and set finalizer")

			ls := &lsv1alpha1.Landscaper{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			}

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(ls), ls)).To(Succeed())
			Expect(ls.Status.ProviderConfigRef.Name).To(Equal("test"))
		})

		It("should install/uninstall a landscaper instance", func() {
			req := reconcile.Request{
				NamespacedName: client.ObjectKey{
					Name:      "test",
					Namespace: "default",
				},
			}

			requestNamespace, err := libutils.StableMCPNamespace(req.Name, req.Namespace)
			Expect(err).NotTo(HaveOccurred())
			requestNameMCP := clusteraccess.StableRequestName(controllerName, req) + "--mcp"
			requestNameWorkload := clusteraccess.StableRequestName(controllerName, req) + "--wl"

			accessRequestMCP := &clustersv1alpha1.AccessRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      requestNameMCP,
					Namespace: requestNamespace,
				},
			}

			workloadClusterRequest := &clustersv1alpha1.ClusterRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      requestNameWorkload,
					Namespace: requestNamespace,
				},
			}

			workloadAccessRequest := &clustersv1alpha1.AccessRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      requestNameWorkload,
					Namespace: requestNamespace,
				},
			}

			ls := &lsv1alpha1.Landscaper{
				ObjectMeta: metav1.ObjectMeta{
					Name:      req.Name,
					Namespace: req.Namespace,
				},
			}

			identity.SetInstanceID(ls, identity.ComputeInstanceID(ls))

			instance := identity.Instance(identity.GetInstanceID(ls))
			installationNamespace := instance.Namespace()

			lsControllerDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "landscaper-controller",
					Namespace: installationNamespace,
				},
			}

			lsMainDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "landscaper-controller-main",
					Namespace: installationNamespace,
				},
			}

			lsWebhooksServerDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "landscaper-webhooks-server",
					Namespace: installationNamespace,
				},
			}

			manifestDeployerDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "manifest-deployer",
					Namespace: installationNamespace,
				},
			}

			helmDeployerDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "helm-deployer",
					Namespace: installationNamespace,
				},
			}

			env := buildTestEnvironmentReconcile("test-03",
				accessRequestMCP,
				workloadClusterRequest,
				workloadAccessRequest,
				lsControllerDeployment,
				lsMainDeployment,
				lsWebhooksServerDeployment,
				manifestDeployerDeployment,
				helmDeployerDeployment)

			reconcileResult := env.ShouldReconcile(req, "reconcile should return a requeue time")
			Expect(reconcileResult.RequeueAfter).ToNot(BeZero())

			// create the namespace for the cluster and access requests
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: requestNamespace,
				},
			}
			Expect(env.Client().Create(env.Ctx, ns)).To(Succeed())

			// now waiting for the MCP access request to be granted
			reconcileResult = env.ShouldReconcile(req, "reconcile should return a requeue time")
			Expect(reconcileResult.RequeueAfter).ToNot(BeZero())

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(accessRequestMCP), accessRequestMCP)).To(Succeed())

			accessRequestMCP.Status.Phase = clustersv1alpha1.REQUEST_GRANTED
			accessRequestMCP.Status.SecretRef = &commonapi.ObjectReference{
				Name:      "access",
				Namespace: requestNamespace,
			}

			Expect(env.Client().Status().Update(env.Ctx, accessRequestMCP)).To(Succeed())

			// now wait for the workload cluster request to be granted
			reconcileResult = env.ShouldReconcile(req, "reconcile should return a requeue time")
			Expect(reconcileResult.RequeueAfter).ToNot(BeZero())

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(workloadClusterRequest), workloadClusterRequest)).To(Succeed())

			workloadClusterRequest.Status.Phase = clustersv1alpha1.REQUEST_GRANTED

			Expect(env.Client().Status().Update(env.Ctx, workloadClusterRequest)).To(Succeed())
			Expect(reconcileResult.RequeueAfter).ToNot(BeZero())

			// now wait for the workload access request to be granted
			reconcileResult = env.ShouldReconcile(req, "reconcile should return a requeue time")
			Expect(reconcileResult.RequeueAfter).ToNot(BeZero())

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(workloadAccessRequest), workloadAccessRequest)).To(Succeed())

			workloadAccessRequest.Status.Phase = clustersv1alpha1.REQUEST_GRANTED
			workloadAccessRequest.Status.SecretRef = &commonapi.ObjectReference{
				Name:      "access",
				Namespace: requestNamespace,
			}

			Expect(env.Client().Status().Update(env.Ctx, workloadAccessRequest)).To(Succeed())

			// now the landscaper should be installed and wait for readiness check
			reconcileResult = env.ShouldReconcile(req, "reconcile should not return a requeue time")
			Expect(reconcileResult.RequeueAfter).ToNot(BeZero())

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(ls), ls)).To(Succeed())
			Expect(ls.Status.Conditions).To(HaveLen(2))
			Expect(ls.Status.Conditions[0].Type).To(Equal(lsv1alpha1.ConditionTypeInstalled))
			Expect(ls.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(ls.Status.Conditions[1].Type).To(Equal(lsv1alpha1.ConditionTypeReady))
			Expect(ls.Status.Conditions[1].Status).To(Equal(metav1.ConditionFalse))

			installationNs := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: installationNamespace,
				},
			}

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(installationNs), installationNs)).To(Succeed())

			// set deployments to ready
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(lsControllerDeployment), lsControllerDeployment)).To(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(lsMainDeployment), lsMainDeployment)).To(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(lsWebhooksServerDeployment), lsWebhooksServerDeployment)).To(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(manifestDeployerDeployment), manifestDeployerDeployment)).To(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(helmDeployerDeployment), helmDeployerDeployment)).To(Succeed())

			setDeploymentReady(env.Ctx, lsControllerDeployment, env.Client())
			setDeploymentReady(env.Ctx, lsMainDeployment, env.Client())
			setDeploymentReady(env.Ctx, lsWebhooksServerDeployment, env.Client())
			setDeploymentReady(env.Ctx, manifestDeployerDeployment, env.Client())
			setDeploymentReady(env.Ctx, helmDeployerDeployment, env.Client())

			// now the landscaper should be ready
			reconcileResult = env.ShouldReconcile(req, "reconcile should not return a requeue time")

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(ls), ls)).To(Succeed())
			Expect(ls.Status.Conditions).To(HaveLen(2))
			Expect(ls.Status.Conditions[0].Type).To(Equal(lsv1alpha1.ConditionTypeInstalled))
			Expect(ls.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(ls.Status.Conditions[1].Type).To(Equal(lsv1alpha1.ConditionTypeReady))
			Expect(ls.Status.Conditions[1].Status).To(Equal(metav1.ConditionTrue))
			Expect(ls.Status.Phase).To(Equal(lsv1alpha1.PhaseReady))

			// delete the landscaper instance
			Expect(env.Client().Delete(env.Ctx, ls)).To(Succeed())
			reconcileResult = env.ShouldReconcile(req, "reconcile should not return a requeue time")
			Expect(reconcileResult.RequeueAfter).To(BeZero())

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(lsControllerDeployment), lsControllerDeployment)).ToNot(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(lsMainDeployment), lsMainDeployment)).ToNot(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(lsWebhooksServerDeployment), lsWebhooksServerDeployment)).ToNot(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(manifestDeployerDeployment), manifestDeployerDeployment)).ToNot(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(helmDeployerDeployment), helmDeployerDeployment)).ToNot(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(installationNs), installationNs)).ToNot(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(accessRequestMCP), accessRequestMCP)).ToNot(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(workloadClusterRequest), accessRequestMCP)).ToNot(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(workloadAccessRequest), accessRequestMCP)).ToNot(Succeed())
		})
	})
})
