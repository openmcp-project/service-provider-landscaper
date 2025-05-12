package helmdeployer

import (
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

func newServiceAccountMutator(h *valuesHelper) resources.Mutator[*core.ServiceAccount] {
	return resources.NewServiceAccountMutator(
		h.helmDeployerComponent.NamespacedDefaultResourceName(),
		h.hostNamespace(),
		h.helmDeployerComponent.Labels(),
		nil)
}

func newClusterRoleBindingMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRoleBinding] {
	return resources.NewClusterRoleBindingMutator(
		h.helmDeployerComponent.ClusterScopedDefaultResourceName(),
		[]rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      h.helmDeployerComponent.NamespacedDefaultResourceName(),
				Namespace: h.hostNamespace(),
			},
		},
		resources.NewClusterRoleRef(h.helmDeployerComponent.ClusterScopedDefaultResourceName()),
		h.helmDeployerComponent.Labels(),
		nil)
}

func newClusterRoleMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRole] {
	return resources.NewClusterRoleMutator(
		h.helmDeployerComponent.ClusterScopedDefaultResourceName(),
		[]rbac.PolicyRule{
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
		h.helmDeployerComponent.Labels(),
		nil)
}
