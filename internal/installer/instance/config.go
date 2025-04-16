package instance

import (
	"github.com/openmcp-project/controller-utils/pkg/clusters"
	core "k8s.io/api/core/v1"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/types"
)

type Configuration struct {
	Instance identity.Instance
	Version  string

	HostCluster     *clusters.Cluster
	ResourceCluster *clusters.Cluster

	Landscaper LandscaperConfig

	ManifestDeployer ManifestDeployerConfig

	HelmDeployer HelmDeployerConfig
}

type LandscaperConfig struct {
	Controller     ControllerConfig
	WebhooksServer WebhooksServerConfig
}

type ControllerConfig struct {
	Image         api.ImageConfiguration
	Resources     core.ResourceRequirements
	ResourcesMain core.ResourceRequirements
	HPAMain       types.HPAValues
}

type WebhooksServerConfig struct {
	Image     api.ImageConfiguration
	Resources core.ResourceRequirements
	HPA       types.HPAValues
}

type ManifestDeployerConfig struct {
	Image     api.ImageConfiguration
	Resources core.ResourceRequirements
	HPA       types.HPAValues
}

type HelmDeployerConfig struct {
	Image     api.ImageConfiguration
	Resources core.ResourceRequirements
	HPA       types.HPAValues
}
