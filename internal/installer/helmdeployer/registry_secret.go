package helmdeployer

import (
	"github.com/openmcp-project/controller-utils/pkg/resources"
	v1 "k8s.io/api/core/v1"
)

func newRegistrySecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	return resources.NewSecretMutator(
		b.helmDeployerComponent.NamespacedResourceName("registries"),
		b.hostNamespace(),
		b.registrySecretsData,
		v1.SecretTypeOpaque,
		b.helmDeployerComponent.Labels(),
		nil)
}
