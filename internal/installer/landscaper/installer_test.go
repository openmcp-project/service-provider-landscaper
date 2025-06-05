package landscaper

import (
	"context"
	"os"
	"testing"

	"github.com/openmcp-project/service-provider-landscaper/test/utils"

	"github.com/gardener/landscaper/apis/config/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/providerconfig"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper Controller Installer Test Suite")
}

var _ = XDescribe("Landscaper Controller Installer", func() {

	const instanceID = "test-g23tp"

	It("should install the landscaper controllers", func() {
		ctx := context.Background()

		hostCluster, err := utils.ClusterFromEnv("WORKLOAD_KUBECONFIG_PATH")
		Expect(err).ToNot(HaveOccurred())

		kubeconfig, err := os.ReadFile(os.Getenv("KUBECONFIG"))
		Expect(err).ToNot(HaveOccurred())

		serviceProviderConfig, err := providerconfig.ReadProviderConfig(os.Getenv("SERVICE_PROVIDER_RESOURCE_PATH"))
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance:       instanceID,
			Version:        "v0.127.0",
			HostCluster:    hostCluster,
			VerbosityLevel: "INFO",
			Configuration:  v1alpha1.LandscaperConfiguration{},
			ServiceAccount: &ServiceAccountValues{Create: true},
			Controller: ControllerValues{
				LandscaperKubeconfig: &KubeconfigValues{
					Kubeconfig: string(kubeconfig),
				},
				Image: api.ImageConfiguration{
					Image: serviceProviderConfig.Spec.Deployment.LandscaperController.Image,
				},
				ReplicaCount:  nil,
				Resources:     corev1.ResourceRequirements{},
				ResourcesMain: corev1.ResourceRequirements{},
				Metrics:       nil,
			},
			WebhooksServer: WebhooksServerValues{
				DisableWebhooks: nil,
				LandscaperKubeconfig: &KubeconfigValues{
					Kubeconfig: string(kubeconfig),
				},
				Image: api.ImageConfiguration{
					Image: serviceProviderConfig.Spec.Deployment.LandscaperWebhooksServer.Image,
				},
				ServicePort: 0,
				Ingress:     nil,
			},
			ImagePullSecrets:   nil,
			PodSecurityContext: nil,
			SecurityContext:    nil,
		}

		err = InstallLandscaper(ctx, values)
		Expect(err).ToNot(HaveOccurred())
	})

	XIt("should uninstall the landscaper controllers", func() {
		ctx := context.Background()

		hostCluster, err := utils.ClusterFromEnv("WORKLOAD_KUBECONFIG_PATH")
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance:    instanceID,
			HostCluster: hostCluster,
		}

		err = UninstallLandscaper(ctx, values)
		Expect(err).ToNot(HaveOccurred())
	})

})
