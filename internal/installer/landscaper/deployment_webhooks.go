package landscaper

import (
	"fmt"
	"k8s.io/utils/ptr"
	"strings"

	"github.com/openmcp-project/controller-utils/pkg/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type webhooksDeploymentMutator struct {
	*valuesHelper
	metadata resources.MetadataMutator
}

var _ resources.Mutator[*appsv1.Deployment] = &webhooksDeploymentMutator{}

func newWebhooksDeploymentMutator(h *valuesHelper) resources.Mutator[*appsv1.Deployment] {
	return &webhooksDeploymentMutator{valuesHelper: h, metadata: resources.NewMetadataMutator()}
}

func (m *webhooksDeploymentMutator) String() string {
	return fmt.Sprintf("deployment %s/%s", m.workloadNamespace(), m.landscaperWebhooksFullName())
}

func (m *webhooksDeploymentMutator) Empty() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperWebhooksFullName(),
			Namespace: m.workloadNamespace(),
		},
	}
}

func (m *webhooksDeploymentMutator) MetadataMutator() resources.MetadataMutator {
	return m.metadata
}

func (m *webhooksDeploymentMutator) Mutate(r *appsv1.Deployment) error {
	r.Labels = m.webhooksComponent.Labels()
	r.Spec = appsv1.DeploymentSpec{
		Replicas: m.values.WebhooksServer.ReplicaCount,
		Selector: &metav1.LabelSelector{MatchLabels: m.webhooksComponent.SelectorLabels()},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: m.webhooksComponent.DeploymentTemplateLabels(),
			},
			Spec: corev1.PodSpec{
				AutomountServiceAccountToken: ptr.To(false),
				Volumes:                      m.volumes(),
				Containers:                   m.containers(),
				SecurityContext:              m.values.PodSecurityContext,
				ImagePullSecrets:             m.values.ImagePullSecrets,
				TopologySpreadConstraints:    m.webhooksComponent.TopologySpreadConstraints(),
			},
		},
	}
	return nil
}

func (m *webhooksDeploymentMutator) containers() []corev1.Container {
	c := corev1.Container{}
	c.Name = "landscaper-webhooks"
	c.Image = m.values.WebhooksServer.Image.Image
	c.ImagePullPolicy = corev1.PullIfNotPresent
	c.Args = m.args()
	c.Env = m.env()
	c.Resources = m.values.WebhooksServer.Resources
	c.VolumeMounts = m.volumeMounts()
	c.SecurityContext = m.values.SecurityContext
	return []corev1.Container{c}
}

func (m *webhooksDeploymentMutator) volumes() []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: m.controllerMCPKubeconfigSecretName(),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: m.controllerMCPKubeconfigSecretName(),
				},
			},
		},
	}

	return volumes
}

func (m *webhooksDeploymentMutator) volumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      m.controllerMCPKubeconfigSecretName(),
			MountPath: fmt.Sprint("/app/ls/", m.controllerMCPKubeconfigSecretName()),
		},
	}

	return volumeMounts
}

func (m *webhooksDeploymentMutator) args() []string {
	a := []string{
		fmt.Sprint("--cert-ns=", m.mcpNamespace()),
	}

	if m.values.WebhooksServer.Ingress != nil {
		a = append(a, fmt.Sprintf("--webhook-url=https://%s", m.values.WebhooksServer.Ingress.Host))
	} else {
		a = append(a, fmt.Sprintf("--webhook-url=https://%s.%s:%d", m.landscaperWebhooksFullName(), m.workloadNamespace(), m.values.WebhooksServer.ServicePort))
	}

	if m.values.VerbosityLevel != "" {
		a = append(a, fmt.Sprintf("-v=%s", m.values.VerbosityLevel))
	}

	a = append(a, fmt.Sprintf("--port=%d", m.values.WebhooksServer.ServicePort))

	if len(m.values.WebhooksServer.DisableWebhooks) > 0 {
		a = append(a, fmt.Sprintf("--disable-webhooks=%s", strings.Join(m.values.WebhooksServer.DisableWebhooks, ",")))
	}

	return a
}

func (m *webhooksDeploymentMutator) env() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "KUBECONFIG",
			Value: fmt.Sprint("/app/ls/", m.controllerMCPKubeconfigSecretName(), "/kubeconfig"),
		},
	}
}
