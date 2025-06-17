package landscaper

import (
	"fmt"

	v2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

type centralHPAMutator struct {
	*valuesHelper
	metadata resources.MetadataMutator
}

var _ resources.Mutator[*v2.HorizontalPodAutoscaler] = &centralHPAMutator{}

func newCentralHPAMutator(b *valuesHelper) resources.Mutator[*v2.HorizontalPodAutoscaler] {
	return &centralHPAMutator{valuesHelper: b, metadata: resources.NewMetadataMutator()}
}

func (m *centralHPAMutator) String() string {
	return fmt.Sprintf("hpa %s/%s", m.workloadNamespace(), m.landscaperFullName())
}

func (m *centralHPAMutator) Empty() *v2.HorizontalPodAutoscaler {
	return &v2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperFullName(),
			Namespace: m.workloadNamespace(),
		},
	}
}

func (m *centralHPAMutator) MetadataMutator() resources.MetadataMutator {
	return m.metadata
}

func (m *centralHPAMutator) Mutate(r *v2.HorizontalPodAutoscaler) error {
	r.Labels = m.controllerComponent.Labels()
	r.Spec = v2.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: v2.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       m.landscaperFullName(),
		},
		MinReplicas: ptr.To[int32](1),
		MaxReplicas: 1,
		Metrics: []v2.MetricSpec{
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "cpu",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: ptr.To[int32](80),
					},
				},
			},
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "memory",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: ptr.To[int32](80),
					},
				},
			},
		},
	}
	return nil
}
