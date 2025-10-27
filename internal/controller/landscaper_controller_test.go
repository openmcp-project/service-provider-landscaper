package controller_test

import (
	"context"
	"time"

	libutils "github.com/openmcp-project/openmcp-operator/lib/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/openmcp-project/service-provider-landscaper/internal/dns"

	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"

	testutils "github.com/openmcp-project/controller-utils/pkg/testing"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	deploymentv1alpha1 "github.com/openmcp-project/openmcp-operator/api/provider/v1alpha1"
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

	"github.com/openmcp-project/service-provider-landscaper/api/v1alpha2"

	lscontroller "github.com/openmcp-project/service-provider-landscaper/internal/controller"

	commonapi "github.com/openmcp-project/openmcp-operator/api/common"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
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

// expectImagePullSecretsValid verifies that a deployment has the expected number of image pull secrets
// and that each secret exists in the given namespace with the correct type and data
func expectImagePullSecretsValid(ctx context.Context, c client.Client, deployment *appsv1.Deployment, expectedCount int, namespace string) {
	Expect(deployment.Spec.Template.Spec.ImagePullSecrets).To(HaveLen(expectedCount))
	for _, s := range deployment.Spec.Template.Spec.ImagePullSecrets {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.Name,
				Namespace: namespace,
			},
		}
		Expect(c.Get(ctx, client.ObjectKeyFromObject(secret), secret)).To(Succeed())
		Expect(secret.Type).To(Equal(corev1.SecretTypeDockerConfigJson))
		Expect(secret.Data).To(HaveKey(".dockerconfigjson"))
	}
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
	utilruntime.Must(deploymentv1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1alpha2.AddToScheme(scheme))
	utilruntime.Must(gatewayv1.Install(scheme))
	utilruntime.Must(gatewayv1alpha2.Install(scheme))

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
				ProviderName:      "landscaper",
				ProviderNamespace: "openmcp-system",
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

			ls := &v1alpha2.Landscaper{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			}

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(ls), ls)).To(Succeed())
			Expect(ls.ObjectMeta.Finalizers).To(ContainElement(v1alpha2.LandscaperFinalizer))

			Expect(ls.Status.ProviderConfigRef.Name).To(Equal("default"))
			Expect(ls.Status.Phase).To(Equal(v1alpha2.PhaseProgressing))
			Expect(ls.Status.Conditions).To(HaveLen(2))
			Expect(ls.Status.Conditions[0].Type).To(Equal(v1alpha2.ConditionTypeInstalled))
			Expect(ls.Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(ls.Status.Conditions[1].Type).To(Equal(v1alpha2.ConditionTypeReady))
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

			ls := &v1alpha2.Landscaper{
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

			ls := &v1alpha2.Landscaper{
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

			inst := identity.Instance(identity.GetInstanceID(ls))
			tlsRoute := &gatewayv1alpha2.TLSRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "webhooks-tls",
					Namespace: inst.Namespace(),
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
				helmDeployerDeployment,
				tlsRoute)

			reconcileResult := env.ShouldReconcile(req, "reconcile should return a requeue time")
			Expect(reconcileResult.RequeueAfter).ToNot(BeZero())

			// now waiting for the MCP access request to be granted
			reconcileResult = env.ShouldReconcile(req, "reconcile should return a requeue time")
			Expect(reconcileResult.RequeueAfter).ToNot(BeZero())

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(accessRequestMCP), accessRequestMCP)).To(Succeed())

			accessRequestMCP.Status.Phase = clustersv1alpha1.REQUEST_GRANTED
			accessRequestMCP.Status.SecretRef = &commonapi.LocalObjectReference{
				Name: "access",
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
			workloadAccessRequest.Status.SecretRef = &commonapi.LocalObjectReference{
				Name: "access",
			}

			Expect(env.Client().Status().Update(env.Ctx, workloadAccessRequest)).To(Succeed())

			// now the landscaper should wait for the tls route to be created and ready
			reconcileResult = env.ShouldReconcile(req, "reconcile should not return a requeue time")
			Expect(reconcileResult.RequeueAfter).ToNot(BeZero())

			// set the tls route to ready
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(tlsRoute), tlsRoute)).To(Succeed())
			tlsRoute.Status.Parents = []gatewayv1alpha2.RouteParentStatus{
				{
					ParentRef: gatewayv1alpha2.ParentReference{
						Name:      dns.DefaultGatewayName,
						Namespace: ptr.To(gatewayv1.Namespace(dns.DefaultGatewayNamespace)),
					},
					Conditions: []metav1.Condition{
						{
							Type:   string(gatewayv1alpha2.RouteConditionAccepted),
							Status: metav1.ConditionTrue,
						},
					},
				},
			}
			Expect(env.Client().Status().Update(env.Ctx, tlsRoute)).To(Succeed())

			// now the landscaper should be installed and wait for readiness check
			reconcileResult = env.ShouldReconcile(req, "reconcile should not return a requeue time")
			Expect(reconcileResult.RequeueAfter).ToNot(BeZero())

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(ls), ls)).To(Succeed())
			Expect(ls.Status.Conditions).To(HaveLen(2))
			Expect(ls.Status.Conditions[0].Type).To(Equal(v1alpha2.ConditionTypeInstalled))
			Expect(ls.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(ls.Status.Conditions[1].Type).To(Equal(v1alpha2.ConditionTypeReady))
			Expect(ls.Status.Conditions[1].Status).To(Equal(metav1.ConditionFalse))

			installationNs := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: installationNamespace,
				},
			}

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(installationNs), installationNs)).To(Succeed())

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(lsControllerDeployment), lsControllerDeployment)).To(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(lsMainDeployment), lsMainDeployment)).To(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(lsWebhooksServerDeployment), lsWebhooksServerDeployment)).To(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(manifestDeployerDeployment), manifestDeployerDeployment)).To(Succeed())
			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(helmDeployerDeployment), helmDeployerDeployment)).To(Succeed())

			// expect the image pull secrets to be created in the installation namespace
			expectImagePullSecretsValid(env.Ctx, env.Client(), lsControllerDeployment, 2, installationNamespace)
			expectImagePullSecretsValid(env.Ctx, env.Client(), lsMainDeployment, 2, installationNamespace)
			expectImagePullSecretsValid(env.Ctx, env.Client(), lsWebhooksServerDeployment, 2, installationNamespace)
			expectImagePullSecretsValid(env.Ctx, env.Client(), manifestDeployerDeployment, 2, installationNamespace)
			expectImagePullSecretsValid(env.Ctx, env.Client(), helmDeployerDeployment, 1, installationNamespace)

			// set deployments to ready
			setDeploymentReady(env.Ctx, lsControllerDeployment, env.Client())
			setDeploymentReady(env.Ctx, lsMainDeployment, env.Client())
			setDeploymentReady(env.Ctx, lsWebhooksServerDeployment, env.Client())
			setDeploymentReady(env.Ctx, manifestDeployerDeployment, env.Client())
			setDeploymentReady(env.Ctx, helmDeployerDeployment, env.Client())

			// now the landscaper should be ready
			reconcileResult = env.ShouldReconcile(req, "reconcile should not return a requeue time")

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(ls), ls)).To(Succeed())
			Expect(ls.Status.Conditions).To(HaveLen(2))
			Expect(ls.Status.Conditions[0].Type).To(Equal(v1alpha2.ConditionTypeInstalled))
			Expect(ls.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(ls.Status.Conditions[1].Type).To(Equal(v1alpha2.ConditionTypeReady))
			Expect(ls.Status.Conditions[1].Status).To(Equal(metav1.ConditionTrue))
			Expect(ls.Status.Phase).To(Equal(v1alpha2.PhaseReady))

			// delete the landscaper instance
			Expect(env.Client().Delete(env.Ctx, ls)).To(Succeed())

			Eventually(func(g Gomega) {
				_ = env.ShouldReconcile(req, "should reconcile after deletion")

				g.Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(lsControllerDeployment), lsControllerDeployment)).ToNot(Succeed())
				g.Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(lsMainDeployment), lsMainDeployment)).ToNot(Succeed())
				g.Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(lsWebhooksServerDeployment), lsWebhooksServerDeployment)).ToNot(Succeed())
				g.Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(manifestDeployerDeployment), manifestDeployerDeployment)).ToNot(Succeed())
				g.Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(helmDeployerDeployment), helmDeployerDeployment)).ToNot(Succeed())
				g.Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(installationNs), installationNs)).ToNot(Succeed())
				g.Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(accessRequestMCP), accessRequestMCP)).ToNot(Succeed())
				g.Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(workloadAccessRequest), workloadAccessRequest)).ToNot(Succeed())
				g.Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(workloadClusterRequest), workloadClusterRequest)).ToNot(Succeed())
			}, 10*time.Second, 1*time.Second).Should(Succeed())
		})
	})
})
