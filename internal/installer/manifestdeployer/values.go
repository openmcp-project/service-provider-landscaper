package manifestdeployer

import (
	"fmt"

	"github.com/openmcp-project/controller-utils/pkg/clusters"

	"github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha2"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/types"
)

type Values struct {
	Instance               identity.Instance `json:"instance,omitempty"`
	Version                string            `json:"version,omitempty"`
	WorkloadCluster        *clusters.Cluster
	VerbosityLevel         string                      `json:"verbosityLevel,omitempty"`
	MCPClusterKubeconfig   string                      `json:"mcpClusterKubeconfig,omitempty"`
	Image                  api.ImageConfiguration      `json:"image,omitempty"`
	ImagePullSecrets       []core.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	ReplicaCount           *int32                      `json:"replicaCount,omitempty"`
	Resources              core.ResourceRequirements   `json:"resources,omitempty"`
	PodSecurityContext     *core.PodSecurityContext    `json:"podSecurityContext,omitempty"`
	SecurityContext        *core.SecurityContext       `json:"securityContext,omitempty"`
	Configuration          v1alpha2.Configuration      `json:"configuration,omitempty"`
	WorkloadClientSettings *ClientSettings             `json:"workloadClientSettings,omitempty"`
	MCPClientSettings      *ClientSettings             `json:"mcpClientSettings,omitempty"`
	HPA                    types.HPAValues             `json:"hpa,omitempty"`
}

type ReleaseValues struct {
	Instance string `json:"instance,omitempty"`
}

type KubeconfigValues struct {
	Kubeconfig string `json:"kubeconfig,omitempty"`
	SecretRef  string `json:"secretRef,omitempty"`
}

type ClientSettings struct {
	Burst int32 `json:"burst,omitempty"`
	QPS   int32 `json:"qps,omitempty"`
}

type ServiceAccountValues struct {
	Create bool `json:"create,omitempty"`
}

func (v *Values) Default() error {
	if v.VerbosityLevel == "" {
		v.VerbosityLevel = "info"
	}
	if v.ReplicaCount == nil {
		v.ReplicaCount = ptr.To(int32(1))
	}
	if v.Configuration.APIVersion == "" {
		v.Configuration.APIVersion = "manifest.deployer.landscaper.gardener.cloud/v1alpha2"
	}
	if v.Configuration.Kind == "" {
		v.Configuration.Kind = "Configuration"
	}
	if v.Configuration.Identity == "" {
		// TODO
		v.Configuration.Identity = fmt.Sprintf("manifest-deployer-%s", v.Instance)
	}
	if v.WorkloadClientSettings == nil {
		v.WorkloadClientSettings = &ClientSettings{}
	}
	if v.WorkloadClientSettings.Burst == 0 {
		v.WorkloadClientSettings.Burst = 30
	}
	if v.WorkloadClientSettings.QPS == 0 {
		v.WorkloadClientSettings.QPS = 20
	}
	if v.MCPClientSettings == nil {
		v.MCPClientSettings = &ClientSettings{}
	}
	if v.MCPClientSettings.Burst == 0 {
		v.MCPClientSettings.Burst = 60
	}
	if v.MCPClientSettings.QPS == 0 {
		v.MCPClientSettings.QPS = 40
	}
	if v.Resources.Requests == nil {
		cpu, err := resource.ParseQuantity("100m")
		if err != nil {
			return err
		}
		memory, err := resource.ParseQuantity("100Mi")
		if err != nil {
			return err
		}
		v.Resources.Requests = core.ResourceList{
			core.ResourceCPU:    cpu,
			core.ResourceMemory: memory,
		}
	}
	if v.HPA.MaxReplicas == 0 {
		v.HPA.MaxReplicas = 1
	}
	if v.HPA.AverageCpuUtilization == nil {
		v.HPA.AverageCpuUtilization = ptr.To(int32(80))
	}
	if v.HPA.AverageMemoryUtilization == nil {
		v.HPA.AverageMemoryUtilization = ptr.To(int32(80))
	}

	return nil
}
