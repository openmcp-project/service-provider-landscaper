package landscaper

import (
	"fmt"
	"k8s.io/utils/ptr"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

type centralDeploymentMutator struct {
	*valuesHelper
	metadata resources.MetadataMutator
}

var _ resources.Mutator[*appsv1.Deployment] = &centralDeploymentMutator{}

func newCentralDeploymentMutator(h *valuesHelper) resources.Mutator[*appsv1.Deployment] {
	return &centralDeploymentMutator{valuesHelper: h, metadata: resources.NewMetadataMutator()}
}

func (m *centralDeploymentMutator) String() string {
	return fmt.Sprintf("deployment %s/%s", m.workloadNamespace(), m.landscaperFullName())
}

func (m *centralDeploymentMutator) Empty() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperFullName(),
			Namespace: m.workloadNamespace(),
		},
	}
}

func (m *centralDeploymentMutator) MetadataMutator() resources.MetadataMutator {
	return m.metadata
}

func (m *centralDeploymentMutator) Mutate(r *appsv1.Deployment) error {
	r.Labels = m.controllerComponent.Labels()
	r.Spec = appsv1.DeploymentSpec{
		Replicas: m.values.Controller.ReplicaCount,
		Selector: &metav1.LabelSelector{MatchLabels: m.controllerComponent.SelectorLabels()},
		Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      m.controllerComponent.SelectorLabels(),
				Annotations: m.templateAnnotations(),
			},
			Spec: corev1.PodSpec{
				AutomountServiceAccountToken: ptr.To(false),
				Volumes:                      m.volumes(),
				Containers:                   m.containers(),
				SecurityContext:              m.values.PodSecurityContext,
				ImagePullSecrets:             m.values.ImagePullSecrets,
			},
		},
	}
	return nil
}

func (m *centralDeploymentMutator) templateAnnotations() map[string]string {
	annotations := map[string]string{
		"checksum/config": m.configHash,
	}
	return annotations
}

func (m *centralDeploymentMutator) containers() []corev1.Container {
	c := corev1.Container{}
	c.Name = "landscaper"
	c.Image = m.values.Controller.Image.Image
	c.Args = m.args()
	c.Env = m.env()
	c.Resources = m.values.Controller.Resources
	c.VolumeMounts = m.volumeMounts()
	c.ImagePullPolicy = corev1.PullIfNotPresent
	c.SecurityContext = m.values.SecurityContext
	c.Ports = m.ports()
	return []corev1.Container{c}
}

func (m *centralDeploymentMutator) volumes() []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: "oci-cache",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: m.configSecretName(),
				},
			},
		},
		{
			Name: m.controllerMCPKubeconfigSecretName(),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: m.controllerMCPKubeconfigSecretName(),
				},
			},
		},
		{
			Name: m.controllerWorkloadKubeconfigSecretName(),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: m.controllerWorkloadKubeconfigSecretName(),
				},
			},
		},
	}

	return volumes
}

func (m *centralDeploymentMutator) volumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "oci-cache",
			MountPath: "/app/ls/oci-cache",
		},
		{
			Name:      "config",
			MountPath: "/app/ls/config",
		},
		{
			Name:      m.controllerMCPKubeconfigSecretName(),
			MountPath: fmt.Sprint("/app/ls/", m.controllerMCPKubeconfigSecretName()),
		},
		{
			Name:      m.controllerMCPKubeconfigSecretName(),
			MountPath: fmt.Sprint("/app/ls/", m.controllerMCPKubeconfigSecretName()),
		},
	}

	return volumeMounts
}

func (m *centralDeploymentMutator) args() []string {
	a := []string{
		"--config=/app/ls/config/config.yaml",
		fmt.Sprint("--landscaper-kubeconfig=/app/ls/", m.controllerMCPKubeconfigSecretName(), "/kubeconfig"),
		fmt.Sprint("--kubeconfig=/app/ls/", m.controllerWorkloadKubeconfigSecretName(), "/kubeconfig"),
	}
	if m.values.VerbosityLevel != "" {
		a = append(a, fmt.Sprintf("-v=%s", m.values.VerbosityLevel))
	}

	return a
}

func (m *centralDeploymentMutator) env() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "LANDSCAPER_MODE",
			Value: "central-landscaper",
		},
		{
			Name: "MY_POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "MY_POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "LS_HOST_CLIENT_BURST",
			Value: strconv.FormatInt(int64(m.values.Controller.WorkloadClientSettings.Burst), 10),
		},
		{
			Name:  "LS_HOST_CLIENT_QPS",
			Value: strconv.FormatInt(int64(m.values.Controller.WorkloadClientSettings.QPS), 10),
		},
		{
			Name:  "LS_RESOURCE_CLIENT_BURST",
			Value: strconv.FormatInt(int64(m.values.Controller.MCPClientSettings.Burst), 10),
		},
		{
			Name:  "LS_RESOURCE_CLIENT_QPS",
			Value: strconv.FormatInt(int64(m.values.Controller.MCPClientSettings.QPS), 10),
		},
	}
}

func (m *centralDeploymentMutator) ports() []corev1.ContainerPort {
	if m.values.Controller.Metrics != nil {
		return []corev1.ContainerPort{
			{
				Name:          "metrics",
				ContainerPort: m.values.Controller.Metrics.Port,
			},
		}
	}
	return nil
}
