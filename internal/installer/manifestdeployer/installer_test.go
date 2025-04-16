package manifestdeployer

import (
	"context"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/cluster"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/providerconfig"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Manifest Deployer Installer Test Suite")
}

var _ = XDescribe("Manifest Deployer Installer", func() {

	const id = "test-g23tp"

	It("should install the manifest deployer", func() {
		ctx := context.Background()

		hostCluster, err := cluster.WorkloadCluster()
		Expect(err).ToNot(HaveOccurred())

		kubeconfig, err := os.ReadFile(os.Getenv("KUBECONFIG"))
		Expect(err).ToNot(HaveOccurred())

		serviceProviderConfig, err := providerconfig.ReadProviderConfig(os.Getenv("SERVICE_PROVIDER_RESOURCE_PATH"))
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance:    id,
			Version:     "v0.127.0",
			HostCluster: hostCluster,
			LandscaperClusterKubeconfig: &KubeconfigValues{
				Kubeconfig: string(kubeconfig),
			},
			Image: api.ImageConfiguration{
				Image: serviceProviderConfig.ManifestDeployer.Image,
			},
			ImagePullSecrets:       nil,
			PodSecurityContext:     nil,
			SecurityContext:        nil,
			ServiceAccount:         &ServiceAccountValues{Create: true},
			HostClientSettings:     nil,
			ResourceClientSettings: nil,
		}

		_, err = InstallManifestDeployer(ctx, values)
		Expect(err).ToNot(HaveOccurred())
	})

	XIt("should uninstall the manifest deployer", func() {
		ctx := context.Background()

		hostCluster, err := cluster.WorkloadCluster()
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance:    id,
			HostCluster: hostCluster,
		}

		err = UninstallManifestDeployer(ctx, values)
		Expect(err).ToNot(HaveOccurred())
	})

})
