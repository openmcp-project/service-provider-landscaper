package landscaper

import (
	"time"

	"github.com/openmcp-project/controller-utils/pkg/clusters"

	"github.com/gardener/landscaper/apis/config/v1alpha1"
	lscore "github.com/gardener/landscaper/apis/core/v1alpha1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha2"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/types"
)

type Values struct {
	Instance           identity.Instance `json:"instance,omitempty"`
	Version            string            `json:"version,omitempty"`
	WorkloadCluster    *clusters.Cluster
	VerbosityLevel     string                           `json:"verbosityLevel,omitempty"`
	Configuration      v1alpha1.LandscaperConfiguration `json:"configuration,omitempty"`
	Controller         ControllerValues                 `json:"controller,omitempty"`
	WebhooksServer     WebhooksServerValues             `json:"webhooksServer,omitempty"`
	ImagePullSecrets   []core.LocalObjectReference      `json:"imagePullSecrets,omitempty"`
	PodSecurityContext *core.PodSecurityContext         `json:"podSecurityContext,omitempty"`
	SecurityContext    *core.SecurityContext            `json:"securityContext,omitempty"`
}

type ServiceAccountValues struct {
	Create bool `json:"create,omitempty"`
}

type ControllerValues struct {
	Installations v1alpha1.InstallationsController `json:"installations,omitempty"` // optional, has default value
	Executions    v1alpha1.ExecutionsController    `json:"executions,omitempty"`    // optional, has default value
	DeployItems   v1alpha1.DeployItemsController   `json:"deployItems,omitempty"`   // optional, has default value
	Contexts      v1alpha1.ContextsController      `json:"contexts,omitempty"`      // optional, has default value

	// MCPKubeconfig contains the kubeconfig for the mcp cluster.
	MCPKubeconfig          string                    `json:"mcpKubeconfig,omitempty"`
	WorkloadKubeconfig     string                    `json:"workloadKubeconfig,omitempty"` // optional, has default value
	Service                *ServiceValues            `json:"service,omitempty"`            // optional, has default values
	Image                  api.ImageConfiguration    `json:"image,omitempty"`
	ReplicaCount           *int32                    `json:"replicaCount,omitempty"`
	Resources              core.ResourceRequirements `json:"resources,omitempty"`
	ResourcesMain          core.ResourceRequirements `json:"resourcesMain,omitempty"`
	Metrics                *MetricsValues            `json:"metrics,omitempty"`
	WorkloadClientSettings ClientSettings            `json:"workloadClientSettings,omitempty"` // optional, has default value
	MCPClientSettings      ClientSettings            `json:"mcpClientSettings,omitempty"`      // optional, has default value
	// HPAMain contains the values for the HPA of the main deployment.
	// (There is no configuration for HPACentral, because its values are fix.)
	HPAMain            types.HPAValues                 `json:"hpaMain,omitempty"`            // optional, has default value
	DeployItemTimeouts *v1alpha1.DeployItemTimeouts    `json:"deployItemTimeouts,omitempty"` // optional, has default value
	HealthChecks       *v1alpha1.AdditionalDeployments `json:"healthChecks,omitempty"`       // optional, has default value
}

const (
	allWebhooks         = "all"
	installationWebhook = "installation"
	executionWebhook    = "execution"
	deployitemWebhook   = "deployitem"
)

type WebhooksServerValues struct {
	DisableWebhooks []string `json:"disableWebhooks,omitempty"`
	// MCPKubeconfig contains the kubeconfig for the mcp cluster.
	MCPKubeconfig string                    `json:"mcpKubeconfig,omitempty"`
	Service       *ServiceValues            `json:"service,omitempty"` // optional, has default value
	Image         api.ImageConfiguration    `json:"image,omitempty"`
	ServicePort   int32                     `json:"servicePort,omitempty"`  // required unless DisableWebhooks contains "all"
	ReplicaCount  *int32                    `json:"replicaCount,omitempty"` // optional - has default value
	Ingress       *IngressValues            `json:"ingress,omitempty"`      // optional - if nil, no ingress will be created.
	Resources     core.ResourceRequirements `json:"resources,omitempty"`    // optional - has default value
	HPA           types.HPAValues           `json:"hpa,omitempty"`          // optional - has default value
}

type CommonControllerValues struct {
	Workers int32 `json:"workers,omitempty"`
}

type ServiceValues struct {
	Type string `json:"type,omitempty"`
	Port int32  `json:"port,omitempty"`
}

type IngressValues struct {
	Host      string  `json:"host,omitempty"`
	ClassName *string `json:"className,omitempty"` // optional - if not set, some annotations are omitted.
	DNSClass  string  `json:"dnsClass,omitempty"`
}

type MetricsValues struct {
	Port int32 `json:"port,omitempty"`
}

type ClientSettings struct {
	Burst int32 `json:"burst,omitempty"`
	QPS   int32 `json:"qps,omitempty"`
}

func (v *Values) Default() error {
	if v.Controller.Installations.Workers == 0 {
		v.Controller.Installations.Workers = 30
	}
	if v.Controller.Executions.Workers == 0 {
		v.Controller.Executions.Workers = 30
	}
	if v.Controller.DeployItems.Workers == 0 {
		v.Controller.DeployItems.Workers = 5
	}
	if v.Controller.Contexts.Workers == 0 {
		v.Controller.Contexts.Workers = 5
	}
	v.Controller.Contexts.Config.Default.Disable = false
	v.Controller.Contexts.Config.Default.ExcludedNamespaces = []string{"kube-system"}

	if v.Controller.Service == nil {
		v.Controller.Service = &ServiceValues{}
	}
	if v.Controller.Service.Type == "" {
		v.Controller.Service.Type = "ClusterIP"
	}
	if v.Controller.Service.Port == 0 {
		v.Controller.Service.Port = 80
	}

	if v.Controller.WorkloadClientSettings.Burst == 0 {
		v.Controller.WorkloadClientSettings.Burst = 30
	}
	if v.Controller.WorkloadClientSettings.QPS == 0 {
		v.Controller.WorkloadClientSettings.QPS = 20
	}
	if v.Controller.MCPClientSettings.Burst == 0 {
		v.Controller.MCPClientSettings.Burst = 60
	}
	if v.Controller.MCPClientSettings.QPS == 0 {
		v.Controller.MCPClientSettings.QPS = 40
	}
	if v.Controller.Resources.Requests == nil {
		cpu, err := resource.ParseQuantity("100m")
		if err != nil {
			return err
		}
		memory, err := resource.ParseQuantity("100Mi")
		if err != nil {
			return err
		}
		v.Controller.Resources.Requests = core.ResourceList{
			core.ResourceCPU:    cpu,
			core.ResourceMemory: memory,
		}
	}
	if v.Controller.ResourcesMain.Requests == nil {
		cpu, err := resource.ParseQuantity("300m")
		if err != nil {
			return err
		}
		memory, err := resource.ParseQuantity("300Mi")
		if err != nil {
			return err
		}
		v.Controller.ResourcesMain.Requests = core.ResourceList{
			core.ResourceCPU:    cpu,
			core.ResourceMemory: memory,
		}
	}
	if v.Controller.HPAMain.MaxReplicas == 0 {
		v.Controller.HPAMain.MaxReplicas = 1
	}
	if v.Controller.HPAMain.AverageCpuUtilization == nil {
		v.Controller.HPAMain.AverageCpuUtilization = ptr.To(int32(80))
	}
	if v.Controller.HPAMain.AverageMemoryUtilization == nil {
		v.Controller.HPAMain.AverageMemoryUtilization = ptr.To(int32(80))
	}

	if v.Controller.DeployItemTimeouts == nil {
		v.Controller.DeployItemTimeouts = &v1alpha1.DeployItemTimeouts{
			Pickup: &lscore.Duration{Duration: 60 * time.Minute},
		}
	}
	if v.WebhooksServer.Service == nil {
		v.WebhooksServer.Service = &ServiceValues{}
	}
	if v.WebhooksServer.Service.Type == "" {
		v.WebhooksServer.Service.Type = "ClusterIP"
	}
	if v.WebhooksServer.Service.Port == 0 {
		v.WebhooksServer.Service.Port = 80
	}
	if v.WebhooksServer.ServicePort == 0 {
		v.WebhooksServer.ServicePort = 9443
	}
	if v.WebhooksServer.ReplicaCount == nil {
		v.WebhooksServer.ReplicaCount = ptr.To[int32](2)
	}
	if v.WebhooksServer.Resources.Requests == nil {
		cpu, err := resource.ParseQuantity("100m")
		if err != nil {
			return err
		}
		memory, err := resource.ParseQuantity("100Mi")
		if err != nil {
			return err
		}
		v.WebhooksServer.Resources.Requests = core.ResourceList{
			core.ResourceCPU:    cpu,
			core.ResourceMemory: memory,
		}
	}

	if v.WebhooksServer.HPA.MaxReplicas == 0 {
		v.WebhooksServer.HPA.MaxReplicas = 2
	}
	if v.WebhooksServer.HPA.AverageCpuUtilization == nil {
		v.WebhooksServer.HPA.AverageCpuUtilization = ptr.To(int32(80))
	}
	if v.WebhooksServer.HPA.AverageMemoryUtilization == nil {
		v.WebhooksServer.HPA.AverageMemoryUtilization = ptr.To(int32(80))
	}

	return nil
}
