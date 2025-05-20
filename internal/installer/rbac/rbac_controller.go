package rbac

import (
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

const (
	controllerServiceAccountName = "landscaper-controller"
)

func newControllerServiceAccountMutator(h *valuesHelper) resources.Mutator[*core.ServiceAccount] {
	m := resources.NewServiceAccountMutator(
		controllerServiceAccountName,
		h.resourceNamespace())
	m.MetadataMutator().WithLabels(h.rbacComponent.Labels())
	return m
}

func newControllerClusterRoleBindingMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRoleBinding] {
	m := resources.NewClusterRoleBindingMutator(
		controllerClusterRoleName(h),
		[]rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      controllerServiceAccountName,
				Namespace: h.resourceNamespace(),
			},
		},
		resources.NewClusterRoleRef(controllerClusterRoleName(h)))
	m.MetadataMutator().WithLabels(h.rbacComponent.Labels())
	return m
}

func newControllerClusterRoleMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRole] {
	m := resources.NewClusterRoleMutator(
		controllerClusterRoleName(h),
		[]rbac.PolicyRule{
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{"landscaper.gardener.cloud"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets", "configmaps"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts/token"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"clusterroles", "clusterrolebindings"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"get", "watch", "create", "update", "patch"},
			},
		})
	m.MetadataMutator().WithLabels(h.rbacComponent.Labels())
	return m
}

func controllerClusterRoleName(h *valuesHelper) string {
	return h.rbacComponent.ClusterScopedResourceName("controller")
}
