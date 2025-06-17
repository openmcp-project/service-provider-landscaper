package landscaper

import (
	"github.com/openmcp-project/controller-utils/pkg/resources"
	v1 "k8s.io/api/core/v1"
)

func newControllerKubeconfigSecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	m := resources.NewSecretMutator(
		b.controllerKubeconfigSecretName(),
		b.hostNamespace(),
		map[string][]byte{
			"kubeconfig": []byte(b.values.Controller.MCPKubeconfig.Kubeconfig),
		},
		v1.SecretTypeOpaque)
	m.MetadataMutator().WithLabels(b.controllerComponent.Labels())
	return m
}
