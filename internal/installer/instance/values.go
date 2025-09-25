package instance

import (
	"github.com/gardener/landscaper/apis/config/v1alpha1"
	"k8s.io/utils/ptr"

	"github.com/openmcp-project/service-provider-landscaper/internal/installer/helmdeployer"
	"github.com/openmcp-project/service-provider-landscaper/internal/installer/landscaper"
	"github.com/openmcp-project/service-provider-landscaper/internal/installer/manifestdeployer"
	"github.com/openmcp-project/service-provider-landscaper/internal/installer/rbac"
)

// rbacValues determines the import values for the installation of the rbac resources
func rbacValues(c *Configuration) *rbac.Values {
	return &rbac.Values{
		Instance:        c.Instance,
		Version:         c.Version,
		MCPCluster:      c.MCPCluster,
		WorkloadCluster: c.WorkloadCluster,
	}
}

// manifestDeployerValues determines the import values for the installation of the manifest deployer
func manifestDeployerValues(c *Configuration, kubeconfigs *rbac.Kubeconfigs) *manifestdeployer.Values {
	v := &manifestdeployer.Values{
		Instance:             c.Instance,
		Version:              c.Version,
		WorkloadCluster:      c.WorkloadCluster,
		Image:                c.ManifestDeployer.Image,
		Resources:            c.ManifestDeployer.Resources,
		HPA:                  c.ManifestDeployer.HPA,
		MCPClusterKubeconfig: string(kubeconfigs.MCPCluster),
	}

	return v

}

// helmDeployerValues determines the import values for the installation of the helm deployer
func helmDeployerValues(c *Configuration, kubeconfigs *rbac.Kubeconfigs) *helmdeployer.Values {
	v := &helmdeployer.Values{
		Instance:             c.Instance,
		Version:              c.Version,
		WorkloadCluster:      c.WorkloadCluster,
		Image:                c.HelmDeployer.Image,
		Resources:            c.HelmDeployer.Resources,
		HPA:                  c.HelmDeployer.HPA,
		MCPClusterKubeconfig: string(kubeconfigs.MCPCluster),
	}

	return v
}

// landscaperValues determines the import values for the installation of the landscaper controllers and webhooks server
func landscaperValues(c *Configuration, kubeconfigs *rbac.Kubeconfigs, manifestExports *manifestdeployer.Exports, helmExports *helmdeployer.Exports) *landscaper.Values {
	v := &landscaper.Values{
		Instance:        c.Instance,
		Version:         c.Version,
		WorkloadCluster: c.WorkloadCluster,
		VerbosityLevel:  "INFO",
		Configuration:   v1alpha1.LandscaperConfiguration{},
		Controller: landscaper.ControllerValues{
			MCPKubeconfig:      string(kubeconfigs.MCPCluster),
			WorkloadKubeconfig: string(kubeconfigs.WorkloadCluster),
			Image:              c.Landscaper.Controller.Image,
			ReplicaCount:       ptr.To[int32](1),
			Resources:          c.Landscaper.Controller.Resources,
			ResourcesMain:      c.Landscaper.Controller.ResourcesMain,
			Metrics:            nil,
			HPAMain:            c.Landscaper.Controller.HPAMain,
		},
		WebhooksServer: landscaper.WebhooksServerValues{
			DisableWebhooks: nil,
			MCPKubeconfig:   string(kubeconfigs.MCPCluster),
			Image:           c.Landscaper.WebhooksServer.Image,
			ServicePort:     c.Landscaper.WebhooksServer.ServicePort,
			Service: &landscaper.ServiceValues{
				Port: c.Landscaper.WebhooksServer.ServicePort,
				Name: c.Landscaper.WebhooksServer.ServiceName,
			},
			URL:       c.WorkloadClusterDomain,
			Resources: c.Landscaper.WebhooksServer.Resources,
			HPA:       c.Landscaper.WebhooksServer.HPA,
		},
	}

	// Deployments to be considered by the health checks
	deployments := []string{}
	if manifestExports != nil {
		deployments = append(deployments, manifestExports.DeploymentName)
	}
	if helmExports != nil {
		deployments = append(deployments, helmExports.DeploymentName)
	}
	v.Controller.HealthChecks = &v1alpha1.AdditionalDeployments{
		Deployments: deployments,
	}

	return v
}
