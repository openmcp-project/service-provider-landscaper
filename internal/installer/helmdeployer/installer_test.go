package helmdeployer_test

import (
	"testing"

	"github.com/openmcp-project/service-provider-landscaper/internal/installer/helmdeployer"
	"github.com/openmcp-project/service-provider-landscaper/internal/installer/rbac"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	testutils "github.com/openmcp-project/controller-utils/pkg/testing"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha2 "github.com/openmcp-project/service-provider-landscaper/api/v1alpha2"
)

const (
	version = "v0.135.0"
)

func TestConfig(t *testing.T) {
	rbac.SetKubeconfigAccessor(rbac.TestKubeconfigAccessorImpl)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Helm Deployer Installer Test Suite")
}

func buildTestEnvironment(testdataDir string, objectsWithStatus ...client.Object) *testutils.Environment {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clustersv1alpha1.AddToScheme(scheme))
	utilruntime.Must(lsv1alpha2.AddToScheme(scheme))

	return testutils.NewEnvironmentBuilder().
		WithFakeClient(scheme).
		WithInitObjectPath("testdata", testdataDir).
		WithInitObjects(objectsWithStatus...).
		Build()
}

var _ = Describe("Helm Deployer Installer", func() {
	const instanceID = "test-g23tp"

	It("should install the helm deployer", func() {
		env := buildTestEnvironment("test-01")

		workloadCluster := clusters.NewTestClusterFromClient("workload", env.Client())
		mcpCluster := clusters.NewTestClusterFromClient("mcp", env.Client())

		kubeconfig, err := rbac.TestKubeconfigAccessorImpl(env.Ctx, mcpCluster)
		Expect(err).ToNot(HaveOccurred())

		providerConfig := lsv1alpha2.ProviderConfig{}
		Expect(env.Client().Get(env.Ctx, client.ObjectKey{Name: "default"}, &providerConfig)).To(Succeed())

		values := &helmdeployer.Values{
			Instance:             instanceID,
			Version:              version,
			WorkloadCluster:      workloadCluster,
			MCPClusterKubeconfig: string(kubeconfig),
			Image: lsv1alpha2.ImageConfiguration{
				Image: providerConfig.GetHelmDeployerImageLocation(version),
			},
			ImagePullSecrets:       nil,
			PodSecurityContext:     nil,
			SecurityContext:        nil,
			WorkloadClientSettings: nil,
			MCPClientSettings:      nil,
		}

		_, err = helmdeployer.InstallHelmDeployer(env.Ctx, values)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should uninstall the helm deployer", func() {
		env := buildTestEnvironment("test-01")

		workloadCluster := clusters.NewTestClusterFromClient("workload", env.Client())
		mcpCluster := clusters.NewTestClusterFromClient("mcp", env.Client())

		kubeconfig, err := rbac.TestKubeconfigAccessorImpl(env.Ctx, mcpCluster)
		Expect(err).ToNot(HaveOccurred())

		values := &helmdeployer.Values{
			Instance:             instanceID,
			WorkloadCluster:      workloadCluster,
			MCPClusterKubeconfig: string(kubeconfig),
		}

		Expect(helmdeployer.UninstallHelmDeployer(env.Ctx, values)).ToNot(HaveOccurred())
	})

})
