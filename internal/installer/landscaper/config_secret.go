package landscaper

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openmcp-project/service-provider-landscaper/internal/installer/resources"
)

type configSecretMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*v1.Secret] = &configSecretMutator{}

func newConfigSecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	return &configSecretMutator{valuesHelper: b}
}

func (m *configSecretMutator) String() string {
	return fmt.Sprintf("secret %s/%s", m.hostNamespace(), m.configSecretName())
}

func (m *configSecretMutator) Empty() *v1.Secret {
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.configSecretName(),
			Namespace: m.hostNamespace(),
		},
	}
}

func (m *configSecretMutator) Mutate(r *v1.Secret) error {
	r.ObjectMeta.Labels = m.controllerComponent.Labels()
	r.Data = map[string][]byte{
		"config.yaml": m.valuesHelper.configYaml,
	}
	return nil
}
