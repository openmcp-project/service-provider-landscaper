package helmdeployer

import (
	"fmt"

	v2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

type hpaMutator struct {
	*valuesHelper
	metadata resources.MetadataMutator
}

var _ resources.Mutator[*v2.HorizontalPodAutoscaler] = &hpaMutator{}

func newHPAMutator(b *valuesHelper) resources.Mutator[*v2.HorizontalPodAutoscaler] {
	return &hpaMutator{valuesHelper: b, metadata: resources.NewMetadataMutator()}
}

func (d *hpaMutator) String() string {
	return fmt.Sprintf("hpa %s/%s", d.workloadNamespace(), d.helmDeployerComponent.NamespacedDefaultResourceName())
}

func (d *hpaMutator) Empty() *v2.HorizontalPodAutoscaler {
	return &v2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.helmDeployerComponent.NamespacedDefaultResourceName(),
			Namespace: d.workloadNamespace(),
		},
	}
}

func (d *hpaMutator) MetadataMutator() resources.MetadataMutator {
	return d.metadata
}

func (d *hpaMutator) Mutate(r *v2.HorizontalPodAutoscaler) error {
	r.Labels = d.helmDeployerComponent.Labels()
	r.Spec = v2.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: v2.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       d.helmDeployerComponent.NamespacedDefaultResourceName(),
		},
		MinReplicas: ptr.To[int32](1),
		MaxReplicas: d.values.HPA.MaxReplicas,
		Metrics: []v2.MetricSpec{
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "cpu",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: d.values.HPA.AverageCpuUtilization,
					},
				},
			},
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "memory",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: d.values.HPA.AverageMemoryUtilization,
					},
				},
			},
		},
	}
	return nil
}
