package landscaper

import (
	"fmt"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

type serviceMutator struct {
	*valuesHelper
	metadata resources.MetadataMutator
}

var _ resources.Mutator[*core.Service] = &serviceMutator{}

func newServiceMutator(b *valuesHelper) resources.Mutator[*core.Service] {
	return &serviceMutator{valuesHelper: b, metadata: resources.NewMetadataMutator()}
}

func (m *serviceMutator) String() string {
	return fmt.Sprintf("landscaper service %s/%s", m.workloadNamespace(), m.landscaperFullName())
}

func (m *serviceMutator) Empty() *core.Service {
	return &core.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperFullName(),
			Namespace: m.workloadNamespace(),
		},
	}
}

func (m *serviceMutator) MetadataMutator() resources.MetadataMutator {
	return m.metadata
}

func (m *serviceMutator) Mutate(r *core.Service) error {
	r.Labels = m.controllerComponent.Labels()
	r.Spec = core.ServiceSpec{
		Ports: []core.ServicePort{
			{
				Name:       "http",
				Port:       m.values.Controller.Service.Port,
				TargetPort: intstr.FromString("http"),
				Protocol:   "TCP",
			},
		},
		Selector: m.controllerComponent.SelectorLabels(),
		Type:     core.ServiceType(m.values.Controller.Service.Type),
	}
	return nil
}
