package landscaper

import (
	"github.com/openmcp-project/controller-utils/pkg/resources"
	v1 "k8s.io/api/core/v1"
)

func newWebhooksKubeconfigSecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	m := resources.NewSecretMutator(
		b.webhooksKubeconfigSecretName(),
		b.hostNamespace(),
		map[string][]byte{
			"kubeconfig": []byte(b.values.WebhooksServer.LandscaperKubeconfig.Kubeconfig),
		},
		v1.SecretTypeOpaque)
	m.MetadataMutator().WithLabels(b.webhooksComponent.Labels())
	return m
}
