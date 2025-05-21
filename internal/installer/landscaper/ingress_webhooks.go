package landscaper

import (
	"fmt"

	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

type ingressMutator struct {
	*valuesHelper
	metadata resources.MetadataMutator
}

var _ resources.Mutator[*networking.Ingress] = &ingressMutator{}

func newIngressMutator(b *valuesHelper) resources.Mutator[*networking.Ingress] {
	return &ingressMutator{valuesHelper: b, metadata: resources.NewMetadataMutator()}
}

func (m *ingressMutator) String() string {
	return fmt.Sprintf("landscaper webhooks ingress %s/%s", m.hostNamespace(), m.landscaperWebhooksFullName())
}

func (m *ingressMutator) Empty() *networking.Ingress {
	return &networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperWebhooksFullName(),
			Namespace: m.hostNamespace(),
		},
	}
}

func (m *ingressMutator) MetadataMutator() resources.MetadataMutator {
	return m.metadata
}

func (m *ingressMutator) Mutate(r *networking.Ingress) error {
	r.Labels = m.webhooksComponent.Labels()
	r.Annotations = map[string]string{
		"nginx.ingress.kubernetes.io/ssl-passthrough": "true",
	}
	if m.values.WebhooksServer.Ingress.DNSClass != "" {
		r.Annotations["dns.gardener.cloud/class"] = m.values.WebhooksServer.Ingress.DNSClass
		r.Annotations["dns.gardener.cloud/dnsnames"] = m.values.WebhooksServer.Ingress.Host
	}
	r.Spec = networking.IngressSpec{
		IngressClassName: m.values.WebhooksServer.Ingress.ClassName,
		Rules: []networking.IngressRule{
			{
				Host: m.values.WebhooksServer.Ingress.Host,
				IngressRuleValue: networking.IngressRuleValue{
					HTTP: &networking.HTTPIngressRuleValue{
						Paths: []networking.HTTPIngressPath{
							{
								Path:     "/",
								PathType: ptr.To(networking.PathTypePrefix),
								Backend: networking.IngressBackend{
									Service: &networking.IngressServiceBackend{
										Name: m.landscaperWebhooksFullName(),
										Port: networking.ServiceBackendPort{
											Number: m.values.WebhooksServer.ServicePort,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return nil
}
