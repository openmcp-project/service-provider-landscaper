package rbac

import (
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

const (
	WebhooksServiceAccountName = "landscaper-webhooks"
)

func newWebhooksServiceAccountMutator(h *valuesHelper) resources.Mutator[*core.ServiceAccount] {
	m := resources.NewServiceAccountMutator(
		WebhooksServiceAccountName,
		h.resourceNamespace())
	m.MetadataMutator().WithLabels(h.rbacComponent.Labels())
	return m
}

func newWebhooksClusterRoleBindingMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRoleBinding] {
	m := resources.NewClusterRoleBindingMutator(
		webhooksClusterRoleName(h),
		[]rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      WebhooksServiceAccountName,
				Namespace: h.resourceNamespace(),
			},
		},
		resources.NewClusterRoleRef(webhooksClusterRoleName(h)))
	m.MetadataMutator().WithLabels(h.rbacComponent.Labels())
	return m
}

func newWebhooksClusterRoleMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRole] {
	m := resources.NewClusterRoleMutator(
		webhooksClusterRoleName(h),
		[]rbac.PolicyRule{
			{
				APIGroups: []string{"landscaper.gardener.cloud"},
				Resources: []string{"installations"},
				Verbs:     []string{"list"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{"admissionregistration.k8s.io"},
				Resources: []string{"validatingwebhookconfigurations"},
				Verbs:     []string{"*"},
			},
		})
	m.MetadataMutator().WithLabels(h.rbacComponent.Labels())
	return m
}

func webhooksClusterRoleName(h *valuesHelper) string {
	return h.rbacComponent.ClusterScopedResourceName("webhooks")
}
