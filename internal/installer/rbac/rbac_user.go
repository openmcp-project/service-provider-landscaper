package rbac

import (
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

const (
	userServiceAccountName = "landscaper-user"
)

func newUserServiceAccountMutator(h *valuesHelper) resources.Mutator[*core.ServiceAccount] {
	m := resources.NewServiceAccountMutator(
		userServiceAccountName,
		h.resourceNamespace())
	m.MetadataMutator().WithLabels(h.rbacComponent.Labels())
	return m
}

func newUserClusterRoleBindingMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRoleBinding] {
	m := resources.NewClusterRoleBindingMutator(
		userClusterRoleName(h),
		[]rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      userServiceAccountName,
				Namespace: h.resourceNamespace(),
			},
		},
		resources.NewClusterRoleRef(userClusterRoleName(h)))
	m.MetadataMutator().WithLabels(h.rbacComponent.Labels())
	return m
}

func newUserClusterRoleMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRole] {
	m := resources.NewClusterRoleMutator(
		userClusterRoleName(h),
		[]rbac.PolicyRule{
			{
				APIGroups: []string{"landscaper.gardener.cloud"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces", "secrets", "configmaps"},
				Verbs:     []string{"*"},
			},
		})
	m.MetadataMutator().WithLabels(h.rbacComponent.Labels())
	return m
}

func userClusterRoleName(h *valuesHelper) string {
	return h.rbacComponent.ClusterScopedResourceName("user")
}
