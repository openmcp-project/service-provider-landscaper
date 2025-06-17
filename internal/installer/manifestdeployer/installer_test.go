package manifestdeployer

import (
	"testing"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	testutils "github.com/openmcp-project/controller-utils/pkg/testing"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/openmcp-project/service-provider-landscaper/internal/shared/cluster"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Manifest Deployer Installer Test Suite")
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

var _ = Describe("Manifest Deployer Installer", func() {
	const instanceID = "test-g23tp"

	It("should install the manifest deployer", func() {
		env := buildTestEnvironment("test-01")

		hostCluster := clusters.NewTestClusterFromClient("workload", env.Client())
		landscaperCluster := clusters.NewTestClusterFromClient("mcp", env.Client())

		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "manifest-deployer",
				Namespace: "landscaper-system",
			},
		}
		Expect(env.Client().Create(env.Ctx, sa)).To(Succeed())

		kubeconfig, err := cluster.CreateKubeconfig(env.Ctx, landscaperCluster, sa)
		Expect(err).ToNot(HaveOccurred())

		providerConfig := lsv1alpha1.ProviderConfig{}
		Expect(env.Client().Get(env.Ctx, client.ObjectKey{Name: "default"}, &providerConfig)).To(Succeed())

		values := &Values{
			Instance:        instanceID,
			Version:         "v0.127.0",
			WorkloadCluster: hostCluster,
			MCPClusterKubeconfig: &KubeconfigValues{
				Kubeconfig: string(kubeconfig),
			},
			Image: lsv1alpha1.ImageConfiguration{
				Image: providerConfig.Spec.Deployment.HelmDeployer.Image,
			},
			ImagePullSecrets:       nil,
			PodSecurityContext:     nil,
			SecurityContext:        nil,
			ServiceAccount:         &ServiceAccountValues{Create: true},
			WorkloadClientSettings: nil,
			MCPClientSettings:      nil,
		}

		_, err = InstallManifestDeployer(env.Ctx, values)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should uninstall the manifest deployer", func() {
		env := buildTestEnvironment("test-01")

		hostCluster := clusters.NewTestClusterFromClient("workload", env.Client())
		landscaperCluster := clusters.NewTestClusterFromClient("mcp", env.Client())

		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "manifest-deployer",
				Namespace: "landscaper-system",
			},
		}
		Expect(env.Client().Create(env.Ctx, sa)).To(Succeed())

		kubeconfig, err := cluster.CreateKubeconfig(env.Ctx, landscaperCluster, sa)
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance:        instanceID,
			WorkloadCluster: hostCluster,
			MCPClusterKubeconfig: &KubeconfigValues{
				Kubeconfig: string(kubeconfig),
			},
		}

		Expect(UninstallManifestDeployer(env.Ctx, values)).ToNot(HaveOccurred())
	})

})
