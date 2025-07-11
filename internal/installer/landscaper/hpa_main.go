package landscaper

import (
	"fmt"

	v2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

type mainHPAMutator struct {
	*valuesHelper
	metadata resources.MetadataMutator
}

var _ resources.Mutator[*v2.HorizontalPodAutoscaler] = &mainHPAMutator{}

func newMainHPAMutator(b *valuesHelper) resources.Mutator[*v2.HorizontalPodAutoscaler] {
	return &mainHPAMutator{valuesHelper: b, metadata: resources.NewMetadataMutator()}
}

func (m *mainHPAMutator) String() string {
	return fmt.Sprintf("hpa %s/%s", m.workloadNamespace(), m.landscaperMainFullName())
}

func (m *mainHPAMutator) Empty() *v2.HorizontalPodAutoscaler {
	return &v2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperMainFullName(),
			Namespace: m.workloadNamespace(),
		},
	}
}

func (m *mainHPAMutator) MetadataMutator() resources.MetadataMutator {
	return m.metadata
}

func (m *mainHPAMutator) Mutate(r *v2.HorizontalPodAutoscaler) error {
	r.Labels = m.controllerMainComponent.Labels()
	r.Spec = v2.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: v2.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       m.landscaperMainFullName(),
		},
		MinReplicas: ptr.To[int32](1),
		MaxReplicas: m.values.Controller.HPAMain.MaxReplicas,
		Metrics: []v2.MetricSpec{
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "cpu",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: m.values.Controller.HPAMain.AverageCpuUtilization,
					},
				},
			},
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "memory",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: m.values.Controller.HPAMain.AverageMemoryUtilization,
					},
				},
			},
		},
	}
	return nil
}
