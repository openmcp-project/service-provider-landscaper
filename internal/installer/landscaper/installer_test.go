package landscaper

import (
	"github.com/openmcp-project/controller-utils/pkg/clusters"
	testutils "github.com/openmcp-project/controller-utils/pkg/testing"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/openmcp-project/service-provider-landscaper/internal/shared/cluster"

	"testing"

	"github.com/gardener/landscaper/apis/config/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper Controller Installer Test Suite")
}

func buildTestEnvironment(testdataDir string, objectsWithStatus ...client.Object) *testutils.Environment {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clustersv1alpha1.AddToScheme(scheme))
	utilruntime.Must(lsv1alpha1.AddToScheme(scheme))

	return testutils.NewEnvironmentBuilder().
		WithFakeClient(scheme).
		WithInitObjectPath("testdata", testdataDir).
		WithInitObjects(objectsWithStatus...).
		Build()
}

var _ = Describe("Landscaper Controller Installer", func() {

	const instanceID = "test-g23tp"

	It("should install the landscaper controllers", func() {
		env := buildTestEnvironment("test-01")

		workloadCluster := clusters.NewTestClusterFromClient("workload", env.Client())
		mcpCluster := clusters.NewTestClusterFromClient("mcp", env.Client())

		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "landscaper",
				Namespace: "landscaper-system",
			},
		}
		Expect(env.Client().Create(env.Ctx, sa)).To(Succeed())

		kubeconfig, err := cluster.CreateKubeconfig(env.Ctx, mcpCluster, sa)
		Expect(err).ToNot(HaveOccurred())

		providerConfig := lsv1alpha1.ProviderConfig{}
		Expect(env.Client().Get(env.Ctx, client.ObjectKey{Name: "default"}, &providerConfig)).To(Succeed())

		values := &Values{
			Instance:       instanceID,
			Version:        "v0.127.0",
			HostCluster:    workloadCluster,
			VerbosityLevel: "INFO",
			Configuration:  v1alpha1.LandscaperConfiguration{},
			ServiceAccount: &ServiceAccountValues{Create: true},
			Controller: ControllerValues{
				MCPKubeconfig: &KubeconfigValues{
					Kubeconfig: string(kubeconfig),
				},
				Image: lsv1alpha1.ImageConfiguration{
					Image: providerConfig.Spec.Deployment.LandscaperController.Image,
				},
				ReplicaCount:  nil,
				Resources:     corev1.ResourceRequirements{},
				ResourcesMain: corev1.ResourceRequirements{},
				Metrics:       nil,
			},
			WebhooksServer: WebhooksServerValues{
				DisableWebhooks: nil,
				MCPKubeconfig: &KubeconfigValues{
					Kubeconfig: string(kubeconfig),
				},
				Image: lsv1alpha1.ImageConfiguration{
					Image: providerConfig.Spec.Deployment.LandscaperWebhooksServer.Image,
				},
				ServicePort: 0,
				Ingress:     nil,
			},
			ImagePullSecrets:   nil,
			PodSecurityContext: nil,
			SecurityContext:    nil,
		}

		err = InstallLandscaper(env.Ctx, values)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should uninstall the landscaper controllers", func() {
		env := buildTestEnvironment("test-01")

		workloadCluster := clusters.NewTestClusterFromClient("workload", env.Client())
		mcpCluster := clusters.NewTestClusterFromClient("mcp", env.Client())

		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "helm-deployer",
				Namespace: "landscaper-system",
			},
		}
		Expect(env.Client().Create(env.Ctx, sa)).To(Succeed())

		kubeconfig, err := cluster.CreateKubeconfig(env.Ctx, mcpCluster, sa)
		Expect(err).ToNot(HaveOccurred())

		providerConfig := lsv1alpha1.ProviderConfig{}
		Expect(env.Client().Get(env.Ctx, client.ObjectKey{Name: "default"}, &providerConfig)).To(Succeed())

		values := &Values{
			Instance:    instanceID,
			HostCluster: workloadCluster,
			WebhooksServer: WebhooksServerValues{
				MCPKubeconfig: &KubeconfigValues{
					Kubeconfig: string(kubeconfig),
				},
			},
			Controller: ControllerValues{
				MCPKubeconfig: &KubeconfigValues{
					Kubeconfig: string(kubeconfig),
				},
			},
		}

		Expect(UninstallLandscaper(env.Ctx, values)).ToNot(HaveOccurred())
	})

})
