package instance_test

import (
	"testing"

	"github.com/openmcp-project/service-provider-landscaper/internal/installer/instance"
	"github.com/openmcp-project/service-provider-landscaper/internal/installer/rbac"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	testutils "github.com/openmcp-project/controller-utils/pkg/testing"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha2 "github.com/openmcp-project/service-provider-landscaper/api/v1alpha2"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/types"
)

const (
	version = "v0.135.0"
)

func TestConfig(t *testing.T) {
	rbac.SetKubeconfigAccessor(rbac.TestKubeconfigAccessorImpl)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper Instance Installer Test Suite")
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

func createConfiguration(env *testutils.Environment) *instance.Configuration {
	providerConfig := lsv1alpha2.ProviderConfig{}
	Expect(env.Client().Get(env.Ctx, client.ObjectKey{Name: "default"}, &providerConfig)).To(Succeed())

	return &instance.Configuration{
		Version: version,
		Landscaper: instance.LandscaperConfig{
			Controller: instance.ControllerConfig{
				Image: lsv1alpha2.ImageConfiguration{
					Image: providerConfig.GetLandscaperControllerImageLocation(version),
				},
			},
			WebhooksServer: instance.WebhooksServerConfig{
				Image: lsv1alpha2.ImageConfiguration{
					Image: providerConfig.GetLandscaperWebhooksServerImageLocation(version),
				},
			},
		},
		ManifestDeployer: instance.ManifestDeployerConfig{
			Image: lsv1alpha2.ImageConfiguration{
				Image: providerConfig.GetManifestDeployerImageLocation(version),
			},
		},
		HelmDeployer: instance.HelmDeployerConfig{
			Image: lsv1alpha2.ImageConfiguration{
				Image: providerConfig.GetHelmDeployerImageLocation(version),
			},
		},
	}
}

var _ = Describe("Landscaper Instance Installer", func() {

	const instanceID = "test2501"

	It("should install the landscaper instance", func() {
		var err error

		env := buildTestEnvironment("test-01")
		config := createConfiguration(env)

		// Create configuration with instance independent values

		// Add instance specific values
		config.Instance = instanceID
		config.WorkloadCluster = clusters.NewTestClusterFromClient("workload", env.Client())
		config.MCPCluster = clusters.NewTestClusterFromClient("mcp", env.Client())

		// Add optional values
		config.HelmDeployer.HPA = types.HPAValues{
			MaxReplicas: 3,
		}
		config.HelmDeployer.Resources = core.ResourceRequirements{
			Requests: map[core.ResourceName]resource.Quantity{
				core.ResourceMemory: resource.MustParse("100Mi"),
			},
		}
		config.Landscaper.Controller.ResourcesMain = core.ResourceRequirements{
			Requests: map[core.ResourceName]resource.Quantity{
				core.ResourceMemory: resource.MustParse("50Mi"),
				core.ResourceCPU:    resource.MustParse("50m"),
			},
		}
		config.Landscaper.WebhooksServer.Resources = core.ResourceRequirements{
			Requests: map[core.ResourceName]resource.Quantity{
				core.ResourceMemory: resource.MustParse("50Mi"),
				core.ResourceCPU:    resource.MustParse("50m"),
			},
		}

		err = instance.InstallLandscaperInstance(env.Ctx, config)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should uninstall the landscaper instance", func() {
		var err error

		// Create configuration with instance independent values
		env := buildTestEnvironment("test-01")
		config := createConfiguration(env)

		Expect(config.Landscaper.Controller.Image.Image).To(Equal("registry.test/" + lsv1alpha2.LandscaperControllerImageLocation + ":" + version))
		Expect(config.Landscaper.WebhooksServer.Image.Image).To(Equal("registry.test/" + lsv1alpha2.LandscaperWebhooksImageLocations + ":" + version))
		Expect(config.ManifestDeployer.Image.Image).To(Equal("registry.test/" + lsv1alpha2.ManifestDeployerImageLocation + ":" + version))
		Expect(config.HelmDeployer.Image.Image).To(Equal("other.registry.test/landscaper/images/helm-deployer-controller" + ":" + version))

		// Add instance specific values
		config.Instance = instanceID
		config.WorkloadCluster = clusters.NewTestClusterFromClient("workload", env.Client())
		config.MCPCluster = clusters.NewTestClusterFromClient("mcp", env.Client())

		err = instance.UninstallLandscaperInstance(env.Ctx, config)
		Expect(err).NotTo(HaveOccurred())
	})

})
