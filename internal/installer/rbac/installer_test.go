package rbac_test

import (
	"testing"

	"github.com/openmcp-project/service-provider-landscaper/internal/installer/rbac"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	testutils "github.com/openmcp-project/controller-utils/pkg/testing"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	lsv1alpha1 "github.com/openmcp-project/service-provider-landscaper/api/v1alpha2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestConfig(t *testing.T) {
	rbac.SetKubeconfigAccessor(rbac.TestKubeconfigAccessorImpl)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper RBAC Installer Test Suite")
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

var _ = Describe("Landscaper RBAC Installer", func() {

	const instanceID = "test-rr8fq"

	It("should install the landscaper rbac resources", func() {
		env := buildTestEnvironment("test-01")

		mcpCluster := clusters.NewTestClusterFromClient("mcp", env.Client())
		workloadCluster := clusters.NewTestClusterFromClient("workload", env.Client())

		values := &rbac.Values{
			Instance:        instanceID,
			Version:         "v0.127.0",
			MCPCluster:      mcpCluster,
			WorkloadCluster: workloadCluster,
		}

		kubeconfigs, err := rbac.GetKubeconfigs(env.Ctx, values)
		Expect(err).ToNot(HaveOccurred())
		err = rbac.InstallLandscaperRBACResources(env.Ctx, values)
		Expect(err).ToNot(HaveOccurred())
		Expect(kubeconfigs.MCPCluster).ToNot(BeEmpty())
		Expect(kubeconfigs.WorkloadCluster).ToNot(BeEmpty())
	})

	It("should uninstall the landscaper rbac resources", func() {
		env := buildTestEnvironment("test-01")

		resourceCluster := clusters.NewTestClusterFromClient("mcp", env.Client())

		values := &rbac.Values{
			Instance:   instanceID,
			MCPCluster: resourceCluster,
		}

		Expect(rbac.UninstallLandscaperRBACResources(env.Ctx, values)).ToNot(HaveOccurred())
	})

})
