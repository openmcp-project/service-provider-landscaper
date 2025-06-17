package landscaper

import (
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

type mainDeploymentMutator struct {
	*valuesHelper
	metadata resources.MetadataMutator
}

var _ resources.Mutator[*appsv1.Deployment] = &mainDeploymentMutator{}

func newMainDeploymentMutator(h *valuesHelper) resources.Mutator[*appsv1.Deployment] {
	return &mainDeploymentMutator{valuesHelper: h, metadata: resources.NewMetadataMutator()}
}

func (m *mainDeploymentMutator) String() string {
	return fmt.Sprintf("deployment %s/%s", m.hostNamespace(), m.landscaperMainFullName())
}

func (m *mainDeploymentMutator) Empty() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperMainFullName(),
			Namespace: m.hostNamespace(),
		},
	}
}

func (m *mainDeploymentMutator) MetadataMutator() resources.MetadataMutator {
	return m.metadata
}

func (m *mainDeploymentMutator) Mutate(r *appsv1.Deployment) error {
	r.Labels = m.controllerMainComponent.Labels()
	r.Spec = appsv1.DeploymentSpec{
		Replicas: m.values.Controller.ReplicaCount,
		Selector: &metav1.LabelSelector{MatchLabels: m.controllerMainComponent.SelectorLabels()},
		Strategy: m.strategy(),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      m.controllerMainComponent.DeploymentTemplateLabels(),
				Annotations: m.templateAnnotations(),
			},
			Spec: corev1.PodSpec{
				Volumes:                   m.volumes(),
				Containers:                m.containers(),
				ServiceAccountName:        m.landscaperFullName(),
				SecurityContext:           m.values.PodSecurityContext,
				ImagePullSecrets:          m.values.ImagePullSecrets,
				TopologySpreadConstraints: m.controllerMainComponent.TopologySpreadConstraints(),
			},
		},
	}
	return nil
}

func (m *mainDeploymentMutator) strategy() appsv1.DeploymentStrategy {
	strategy := appsv1.DeploymentStrategy{}
	if m.values.Controller.HPAMain.MaxReplicas == 1 {
		strategy.Type = appsv1.RecreateDeploymentStrategyType
	}
	return strategy
}

func (m *mainDeploymentMutator) templateAnnotations() map[string]string {
	annotations := map[string]string{
		"checksum/config": m.configHash,
	}
	return annotations
}

func (m *mainDeploymentMutator) containers() []corev1.Container {
	c := corev1.Container{}
	c.Name = "landscaper-main"
	c.Image = m.values.Controller.Image.Image
	c.Args = m.args()
	c.Env = m.env()
	c.Resources = m.values.Controller.ResourcesMain
	c.VolumeMounts = m.volumeMounts()
	c.ImagePullPolicy = corev1.PullIfNotPresent
	c.SecurityContext = m.values.SecurityContext
	c.Ports = m.ports()
	return []corev1.Container{c}
}

func (m *mainDeploymentMutator) volumes() []corev1.Volume {
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
	}

	if k := m.values.Controller.MCPKubeconfig; k != nil {
		secretName := ""
		if k.Kubeconfig != "" {
			secretName = m.controllerKubeconfigSecretName()
		} else {
			secretName = k.SecretRef
		}

		kubeconfigVolume := corev1.Volume{
			Name: "landscaper-cluster-kubeconfig",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		}

		volumes = append(volumes, kubeconfigVolume)
	}

	return volumes
}

func (m *mainDeploymentMutator) volumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "oci-cache",
			MountPath: "/app/ls/oci-cache",
		},
		{
			Name:      "config",
			MountPath: "/app/ls/config",
		},
	}
	if m.values.Controller.MCPKubeconfig != nil {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "landscaper-cluster-kubeconfig",
			MountPath: "/app/ls/landscaper-cluster-kubeconfig",
		})
	}
	return volumeMounts
}

func (m *mainDeploymentMutator) args() []string {
	a := []string{
		"--config=/app/ls/config/config.yaml",
	}
	if m.values.Controller.MCPKubeconfig != nil {
		a = append(a, "--landscaper-kubeconfig=/app/ls/landscaper-cluster-kubeconfig/kubeconfig")
	}
	if m.values.VerbosityLevel != "" {
		a = append(a, fmt.Sprintf("-v=%s", m.values.VerbosityLevel))
	}
	return a
}

func (m *mainDeploymentMutator) env() []corev1.EnvVar {
	return []corev1.EnvVar{
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

func (m *mainDeploymentMutator) ports() []corev1.ContainerPort {
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
