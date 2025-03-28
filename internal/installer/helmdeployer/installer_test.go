package helmdeployer

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
	RunSpecs(t, "Helm Deployer Installer Test Suite")
}

var _ = Describe("Helm Deployer Installer", func() {

	const instanceID = "test-g23tp"

	newHostCluster := func() (*cluster.Cluster, error) {
		return cluster.NewCluster(os.Getenv("KUBECONFIG"))
	}

	It("should install the helm deployer", func() {
		ctx := context.Background()

		hostCluster, err := newHostCluster()
		Expect(err).ToNot(HaveOccurred())

		kubeconfig, err := os.ReadFile(os.Getenv("KUBECONFIG"))
		Expect(err).ToNot(HaveOccurred())

		serviceProviderConfig, err := providerconfig.ReadProviderConfig(os.Getenv("SERVICE_PROVIDER_RESOURCE_PATH"))
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance:    instanceID,
			Version:     "v0.127.0",
			HostCluster: hostCluster,
			LandscaperClusterKubeconfig: &KubeconfigValues{
				Kubeconfig: string(kubeconfig),
			},
			Image: api.ImageConfiguration{
				Image: serviceProviderConfig.HelmDeployer.Image,
			},
			ImagePullSecrets:       nil,
			PodSecurityContext:     nil,
			SecurityContext:        nil,
			ServiceAccount:         &ServiceAccountValues{Create: true},
			HostClientSettings:     nil,
			ResourceClientSettings: nil,
			NodeSelector:           nil,
		}

		_, err = InstallHelmDeployer(ctx, values)
		Expect(err).ToNot(HaveOccurred())
	})

	XIt("should uninstall the helm deployer", func() {
		ctx := context.Background()

		hostCluster, err := newHostCluster()
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance:    instanceID,
			HostCluster: hostCluster,
		}

		err = UninstallHelmDeployer(ctx, values)
		Expect(err).ToNot(HaveOccurred())
	})

})
