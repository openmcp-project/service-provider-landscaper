package rbac

import (
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/cluster"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"
)

type Values struct {
	Instance        identity.Instance `json:"instance,omitempty"`
	Version         string            `json:"version,omitempty"`
	ResourceCluster cluster.Cluster
	ServiceAccount  *ServiceAccountValues `json:"serviceAccount,omitempty"`
}

type ServiceAccountValues struct {
	Create bool `json:"create,omitempty"`
}
