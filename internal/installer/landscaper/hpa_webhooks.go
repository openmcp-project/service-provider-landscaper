package landscaper

import (
	"fmt"

	v2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

type webhooksHPAMutator struct {
	*valuesHelper
	metadata resources.MetadataMutator
}

var _ resources.Mutator[*v2.HorizontalPodAutoscaler] = &webhooksHPAMutator{}

func newWebhooksHPAMutator(b *valuesHelper) resources.Mutator[*v2.HorizontalPodAutoscaler] {
	return &webhooksHPAMutator{valuesHelper: b, metadata: resources.NewMetadataMutator()}
}

func (m *webhooksHPAMutator) String() string {
	return fmt.Sprintf("hpa %s/%s", m.workloadNamespace(), m.landscaperWebhooksFullName())
}

func (m *webhooksHPAMutator) Empty() *v2.HorizontalPodAutoscaler {
	return &v2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperWebhooksFullName(),
			Namespace: m.workloadNamespace(),
		},
	}
}

func (m *webhooksHPAMutator) MetadataMutator() resources.MetadataMutator {
	return m.metadata
}

func (m *webhooksHPAMutator) Mutate(r *v2.HorizontalPodAutoscaler) error {
	r.Labels = m.webhooksComponent.Labels()
	r.Spec = v2.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: v2.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       m.landscaperWebhooksFullName(),
		},
		MinReplicas: ptr.To[int32](2),
		MaxReplicas: m.values.WebhooksServer.HPA.MaxReplicas,
		Metrics: []v2.MetricSpec{
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "cpu",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: m.values.WebhooksServer.HPA.AverageCpuUtilization,
					},
				},
			},
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "memory",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: m.values.WebhooksServer.HPA.AverageMemoryUtilization,
					},
				},
			},
		},
	}
	return nil
}
