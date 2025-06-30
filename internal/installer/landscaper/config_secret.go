package landscaper

import (
	"github.com/openmcp-project/controller-utils/pkg/resources"
	v1 "k8s.io/api/core/v1"
)

func newConfigSecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	m := resources.NewSecretMutator(
		b.configSecretName(),
		b.workloadNamespace(),
		map[string][]byte{
			"config.yaml": b.configYaml,
		},
		v1.SecretTypeOpaque)
	m.MetadataMutator().WithLabels(b.controllerComponent.Labels())
	return m
}
