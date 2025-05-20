package landscaper

import (
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

func newServiceAccountMutator(h *valuesHelper) resources.Mutator[*core.ServiceAccount] {
	m := resources.NewServiceAccountMutator(
		h.landscaperFullName(),
		h.hostNamespace())
	m.MetadataMutator().WithLabels(h.controllerComponent.Labels())
	return m
}

func newClusterRoleBindingMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRoleBinding] {
	m := resources.NewClusterRoleBindingMutator(
		h.controllerComponent.ClusterScopedDefaultResourceName(),
		[]rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      h.landscaperFullName(),
				Namespace: h.hostNamespace(),
			},
		},
		resources.NewClusterRoleRef(h.controllerComponent.ClusterScopedDefaultResourceName()))
	m.MetadataMutator().WithLabels(h.controllerComponent.Labels())
	return m
}

func newClusterRoleMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRole] {
	m := resources.NewClusterRoleMutator(
		h.controllerComponent.ClusterScopedDefaultResourceName(),
		[]rbac.PolicyRule{
			{
				APIGroups: []string{"landscaper.gardener.cloud"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				// The agent contains a helm deployer to deploy other deployers.
				// Its unknown what deployers might need we have to give the agent all possible permissions for resources.
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		})
	m.MetadataMutator().WithLabels(h.controllerComponent.Labels())
	return m
}
