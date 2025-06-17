package helmdeployer

import (
	"github.com/openmcp-project/controller-utils/pkg/resources"
	v1 "k8s.io/api/core/v1"
)

func newRegistrySecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	m := resources.NewSecretMutator(
		b.helmDeployerComponent.NamespacedResourceName("registries"),
		b.workloadNamespace(),
		b.registrySecretsData,
		v1.SecretTypeOpaque)
	m.MetadataMutator().WithLabels(b.helmDeployerComponent.Labels())
	return m
}
