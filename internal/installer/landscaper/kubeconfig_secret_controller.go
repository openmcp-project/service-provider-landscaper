package landscaper

import (
	"github.com/openmcp-project/controller-utils/pkg/resources"
	v1 "k8s.io/api/core/v1"
)

func newControllerMCPKubeconfigSecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	m := resources.NewSecretMutator(
		b.controllerMCPKubeconfigSecretName(),
		b.workloadNamespace(),
		map[string][]byte{
			"kubeconfig": []byte(b.values.Controller.MCPKubeconfig),
		},
		v1.SecretTypeOpaque)
	m.MetadataMutator().WithLabels(b.controllerComponent.Labels())
	return m
}

func newControllerWorkloadKubeconfigSecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	m := resources.NewSecretMutator(
		b.controllerWorkloadKubeconfigSecretName(),
		b.workloadNamespace(),
		map[string][]byte{
			"kubeconfig": []byte(b.values.Controller.WorkloadKubeconfig),
		},
		v1.SecretTypeOpaque)
	m.MetadataMutator().WithLabels(b.controllerComponent.Labels())
	return m
}
