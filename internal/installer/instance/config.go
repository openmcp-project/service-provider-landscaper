package instance

import (
	core "k8s.io/api/core/v1"

	"github.com/openmcp-project/controller-utils/pkg/clusters"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha2"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/types"
)

type Configuration struct {
	Instance identity.Instance
	Version  string

	MCPCluster            *clusters.Cluster
	WorkloadCluster       *clusters.Cluster
	WorkloadClusterDomain string

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
	Image       api.ImageConfiguration
	Resources   core.ResourceRequirements
	HPA         types.HPAValues
	ServiceName string
	ServicePort int32
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
