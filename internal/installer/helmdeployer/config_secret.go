package helmdeployer

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

func (d *configSecretMutator) String() string {
	return fmt.Sprintf("config secret %s/%s", d.hostNamespace(), d.name())
}

func (d *configSecretMutator) name() string {
	return d.helmDeployerComponent.NamespacedResourceName("config")
}

func (d *configSecretMutator) Empty() *v1.Secret {
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.name(),
			Namespace: d.hostNamespace(),
		},
	}
}

func (d *configSecretMutator) Mutate(r *v1.Secret) error {
	r.ObjectMeta.Labels = d.helmDeployerComponent.Labels()
	r.Data = map[string][]byte{
		"config.yaml": d.configYaml,
	}
	return nil
}
