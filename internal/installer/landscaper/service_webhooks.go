package landscaper

import (
	"fmt"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

type webhooksServiceMutator struct {
	*valuesHelper
	metadata resources.MetadataMutator
}

var _ resources.Mutator[*core.Service] = &webhooksServiceMutator{}

func newWebhooksServiceMutator(b *valuesHelper) resources.Mutator[*core.Service] {
	return &webhooksServiceMutator{valuesHelper: b, metadata: resources.NewMetadataMutator()}
}

func (m *webhooksServiceMutator) String() string {
	return fmt.Sprintf("landscaper webhooks service %s/%s", m.workloadNamespace(), m.landscaperWebhooksFullName())
}

func (m *webhooksServiceMutator) MetadataMutator() resources.MetadataMutator {
	return m.metadata
}

func (m *webhooksServiceMutator) Empty() *core.Service {
	return &core.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperWebhooksFullName(),
			Namespace: m.workloadNamespace(),
		},
	}
}

func (m *webhooksServiceMutator) Mutate(r *core.Service) error {
	r.Labels = m.webhooksComponent.Labels()
	r.Spec = core.ServiceSpec{
		Ports: []core.ServicePort{
			{
				Name:       "webhooks",
				Port:       m.values.WebhooksServer.ServicePort,
				TargetPort: intstr.FromInt32(m.values.WebhooksServer.ServicePort),
				Protocol:   "TCP",
			},
		},
		Selector: m.webhooksComponent.SelectorLabels(),
		Type:     core.ServiceType(m.values.WebhooksServer.Service.Type),
	}
	return nil
}
