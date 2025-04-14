package helmdeployer

import (
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"

	"github.com/openmcp-project/service-provider-landscaper/internal/shared/resources"
)

func newServiceAccountMutator(h *valuesHelper) resources.Mutator[*core.ServiceAccount] {
	return &resources.ServiceAccountMutator{
		Name:      h.helmDeployerComponent.NamespacedDefaultResourceName(),
		Namespace: h.hostNamespace(),
		Labels:    h.helmDeployerComponent.Labels(),
	}
}

func newClusterRoleBindingMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRoleBinding] {
	return &resources.ClusterRoleBindingMutator{
		ClusterRoleBindingName:  h.helmDeployerComponent.ClusterScopedDefaultResourceName(),
		ClusterRoleName:         h.helmDeployerComponent.ClusterScopedDefaultResourceName(),
		ServiceAccountName:      h.helmDeployerComponent.NamespacedDefaultResourceName(),
		ServiceAccountNamespace: h.hostNamespace(),
		Labels:                  h.helmDeployerComponent.Labels(),
	}
}

func newClusterRoleMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRole] {
	return &resources.ClusterRoleMutator{
		Name:   h.helmDeployerComponent.ClusterScopedDefaultResourceName(),
		Labels: h.helmDeployerComponent.Labels(),
		Rules: []rbac.PolicyRule{
			{
				APIGroups: []string{"landscaper.gardener.cloud"},
				Resources: []string{"deployitems", "deployitems/status"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
			{
				APIGroups: []string{"landscaper.gardener.cloud"},
				Resources: []string{"targets", "contexts"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"landscaper.gardener.cloud"},
				Resources: []string{"syncobjects", "criticalproblems"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces", "pods", "configmaps"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts/token"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"get", "watch", "create", "update", "patch"},
			},
		},
	}
}
