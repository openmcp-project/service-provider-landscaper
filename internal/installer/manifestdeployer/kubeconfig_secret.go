package manifestdeployer

import (
	"github.com/openmcp-project/controller-utils/pkg/resources"
	v1 "k8s.io/api/core/v1"
)

func newKubeconfigSecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	return resources.NewSecretMutator(
		b.manifestDeployerComponent.NamespacedResourceName("landscaper-cluster-kubeconfig"),
		b.hostNamespace(),
		map[string][]byte{
			"kubeconfig": b.landscaperClusterKubeconfig(),
		},
		v1.SecretTypeOpaque,
		b.manifestDeployerComponent.Labels(),
		nil)
}
