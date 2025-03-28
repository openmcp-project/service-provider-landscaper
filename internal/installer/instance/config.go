package instance

import (
	core "k8s.io/api/core/v1"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
	"github.com/openmcp-project/service-provider-landscaper/internal/installer/shared"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/cluster"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"
)

const (
	helm     = "helm"
	manifest = "manifest"
)

type Configuration struct {
	Instance identity.Instance
	Version  string

	HostCluster     *cluster.Cluster
	ResourceCluster *cluster.Cluster

	// Deployers is the list of deployers that are getting installed alongside with this Instance.
	// Supported deployers are: "helm", "manifest".
	Deployers []string

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
	HPAMain       shared.HPAValues
}

type WebhooksServerConfig struct {
	Image     api.ImageConfiguration
	Resources core.ResourceRequirements
	HPA       shared.HPAValues
}

type ManifestDeployerConfig struct {
	Image     api.ImageConfiguration
	Resources core.ResourceRequirements
	HPA       shared.HPAValues
}

type HelmDeployerConfig struct {
	Image     api.ImageConfiguration
	Resources core.ResourceRequirements
	HPA       shared.HPAValues
}
