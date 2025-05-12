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
	return resources.NewServiceAccountMutator(
		userServiceAccountName,
		h.resourceNamespace(),
		h.rbacComponent.Labels(),
		nil)
}

func newUserClusterRoleBindingMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRoleBinding] {
	return resources.NewClusterRoleBindingMutator(
		userClusterRoleName(h),
		[]rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      userServiceAccountName,
				Namespace: h.resourceNamespace(),
			},
		},
		resources.NewClusterRoleRef(userClusterRoleName(h)),
		h.rbacComponent.Labels(),
		nil)
}

func newUserClusterRoleMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRole] {
	return resources.NewClusterRoleMutator(
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
		},
		h.rbacComponent.Labels(),
		nil,
	)
}

func userClusterRoleName(h *valuesHelper) string {
	return h.rbacComponent.ClusterScopedResourceName("user")
}
