package instance

import (
	"context"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/cluster"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/providerconfig"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/types"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper Instance Installer Test Suite")
}

var _ = XDescribe("Landscaper Instance Installer", func() {

	const instanceID = "test2501"

	// newConfiguration creates a  Configuration which is partially filled, namely with the instance independent values.
	newConfiguration := func() (*Configuration, error) {
		serviceProviderConfig, err := providerconfig.ReadProviderConfig(os.Getenv("SERVICE_PROVIDER_RESOURCE_PATH"))
		if err != nil {
			return nil, err
		}

		return &Configuration{
			Version: "v0.127.0",
			Landscaper: LandscaperConfig{
				Controller: ControllerConfig{
					Image: api.ImageConfiguration{
						Image: serviceProviderConfig.Spec.Deployment.LandscaperController.Image,
					},
				},
				WebhooksServer: WebhooksServerConfig{
					Image: api.ImageConfiguration{
						Image: serviceProviderConfig.Spec.Deployment.LandscaperWebhooksServer.Image,
					},
				},
			},
			ManifestDeployer: ManifestDeployerConfig{
				Image: api.ImageConfiguration{
					Image: serviceProviderConfig.Spec.Deployment.ManifestDeployer.Image,
				},
			},
			HelmDeployer: HelmDeployerConfig{
				Image: api.ImageConfiguration{
					Image: serviceProviderConfig.Spec.Deployment.HelmDeployer.Image,
				},
			},
		}, nil
	}

	It("should install the landscaper instance", func() {
		var err error
		ctx := context.Background()

		// Create configuration with instance independent values
		config, err := newConfiguration()
		Expect(err).NotTo(HaveOccurred())

		// Add instance specific values
		config.Instance = instanceID
		config.HostCluster, err = cluster.WorkloadCluster()
		Expect(err).NotTo(HaveOccurred())
		config.ResourceCluster, err = cluster.MCPCluster(ctx, client.ObjectKey{}, nil)
		Expect(err).NotTo(HaveOccurred())

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

		err = InstallLandscaperInstance(ctx, config)
		Expect(err).NotTo(HaveOccurred())
	})

	XIt("should uninstall the landscaper instance", func() {
		var err error
		ctx := context.Background()

		// Create configuration with instance independent values
		config, err := newConfiguration()
		Expect(err).NotTo(HaveOccurred())

		// Add instance specific values
		config.Instance = instanceID
		config.HostCluster, err = cluster.WorkloadCluster()
		Expect(err).NotTo(HaveOccurred())
		config.ResourceCluster, err = cluster.MCPCluster(ctx, client.ObjectKey{}, nil)
		Expect(err).NotTo(HaveOccurred())

		err = UninstallLandscaperInstance(ctx, config)
		Expect(err).NotTo(HaveOccurred())
	})

})
