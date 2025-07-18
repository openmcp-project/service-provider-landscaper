package helmdeployer

import (
	"fmt"
	"strconv"

	"k8s.io/utils/ptr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

type deploymentMutator struct {
	*valuesHelper
	metadata resources.MetadataMutator
}

var _ resources.Mutator[*appsv1.Deployment] = &deploymentMutator{}

func newDeploymentMutator(b *valuesHelper) resources.Mutator[*appsv1.Deployment] {
	return &deploymentMutator{valuesHelper: b, metadata: resources.NewMetadataMutator()}
}

func (d *deploymentMutator) String() string {
	return fmt.Sprintf("deployment %s/%s", d.workloadNamespace(), d.helmDeployerComponent.NamespacedDefaultResourceName())
}

func (d *deploymentMutator) Empty() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.helmDeployerComponent.NamespacedDefaultResourceName(),
			Namespace: d.workloadNamespace(),
		},
	}
}

func (d *deploymentMutator) MetadataMutator() resources.MetadataMutator {
	return d.metadata
}

func (d *deploymentMutator) Mutate(r *appsv1.Deployment) error {
	r.Labels = d.helmDeployerComponent.Labels()
	r.Spec = appsv1.DeploymentSpec{
		Replicas: d.values.ReplicaCount,
		Selector: &metav1.LabelSelector{MatchLabels: d.helmDeployerComponent.SelectorLabels()},
		Strategy: d.strategy(),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      d.helmDeployerComponent.DeploymentTemplateLabels(),
				Annotations: d.templateAnnotations(),
			},

			Spec: corev1.PodSpec{
				AutomountServiceAccountToken: ptr.To(false),
				Volumes:                      d.volumes(),
				Containers:                   d.containers(),
				SecurityContext:              d.values.PodSecurityContext,
				ImagePullSecrets:             d.values.ImagePullSecrets,
				TopologySpreadConstraints:    d.helmDeployerComponent.TopologySpreadConstraints(),
			},
		},
	}
	return nil
}

func (d *deploymentMutator) strategy() appsv1.DeploymentStrategy {
	strategy := appsv1.DeploymentStrategy{}
	if d.values.HPA.MaxReplicas == 1 {
		strategy.Type = appsv1.RecreateDeploymentStrategyType
	}
	return strategy
}

func (d *deploymentMutator) templateAnnotations() map[string]string {
	annotations := map[string]string{
		"checksum/config":          d.configHash,
		"checksum/registrysecrets": d.registrySecretsHash,
	}
	return annotations
}

func (d *deploymentMutator) containers() []corev1.Container {
	c := corev1.Container{}
	c.Name = "helm-deployer"
	c.Image = d.values.Image.Image
	c.Args = d.args()
	c.Env = d.env()
	c.Resources = d.values.Resources
	c.VolumeMounts = d.volumeMounts()
	c.ImagePullPolicy = corev1.PullIfNotPresent
	c.SecurityContext = d.values.SecurityContext
	return []corev1.Container{c}
}

func (d *deploymentMutator) volumes() []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: fmt.Sprintf("%s-config", d.helmDeployerComponent.NamespacedDefaultResourceName()),
				},
			},
		},
		{
			Name: d.mcpKubeconfigSecretName(),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: d.mcpKubeconfigSecretName(),
				},
			},
		},
	}

	if d.values.OCI != nil {
		ociRegistryVolume := corev1.Volume{
			Name: "ociregistry",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: fmt.Sprintf("%s-registries", d.helmDeployerComponent.NamespacedDefaultResourceName()),
				},
			},
		}

		volumes = append(volumes, ociRegistryVolume)
	}

	return volumes
}

func (d *deploymentMutator) volumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/app/ls/config",
		},
		{
			Name:      d.mcpKubeconfigSecretName(),
			MountPath: fmt.Sprint("/app/ls/", d.mcpKubeconfigSecretName()),
		},
	}

	if d.values.OCI != nil {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "ociregistry",
			MountPath: "/app/ls/registry/secrets",
		})
	}

	return volumeMounts
}

func (d *deploymentMutator) args() []string {
	a := []string{
		"--config=/app/ls/config/config.yaml",
	}
	if d.values.VerbosityLevel != "" {
		a = append(a, fmt.Sprintf("-v=%s", d.values.VerbosityLevel))
	}
	return a
}

func (d *deploymentMutator) env() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "KUBECONFIG",
			Value: fmt.Sprintf("/app/ls/%s/kubeconfig", d.mcpKubeconfigSecretName()),
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
			Value: strconv.FormatInt(int64(d.values.WorkloadClientSettings.Burst), 10),
		},
		{
			Name:  "LS_HOST_CLIENT_QPS",
			Value: strconv.FormatInt(int64(d.values.WorkloadClientSettings.QPS), 10),
		},
		{
			Name:  "LS_RESOURCE_CLIENT_BURST",
			Value: strconv.FormatInt(int64(d.values.MCPClientSettings.Burst), 10),
		},
		{
			Name:  "LS_RESOURCE_CLIENT_QPS",
			Value: strconv.FormatInt(int64(d.values.MCPClientSettings.QPS), 10),
		},
	}
}
