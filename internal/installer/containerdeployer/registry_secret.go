package containerdeployer

import (
	"github.com/openmcp-project/controller-utils/pkg/resources"
	v1 "k8s.io/api/core/v1"
)

func newRegistrySecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	m := resources.NewSecretMutator(
		b.containerDeployerComponent.NamespacedResourceName("registries"),
		b.workloadNamespace(),
		b.registrySecretsData,
		v1.SecretTypeOpaque)
	m.MetadataMutator().WithLabels(b.containerDeployerComponent.Labels())
	return m
}
