package rbac

import (
	"github.com/openmcp-project/controller-utils/pkg/clusters"

	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"
)

type Values struct {
	Instance        identity.Instance `json:"instance,omitempty"`
	Version         string            `json:"version,omitempty"`
	MCPCluster      *clusters.Cluster
	WorkloadCluster *clusters.Cluster `json:"workloadCluster,omitempty"`
}

type ServiceAccountValues struct {
	Create bool `json:"create,omitempty"`
}
