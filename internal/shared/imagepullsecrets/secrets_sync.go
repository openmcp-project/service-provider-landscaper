package imagepullsecrets

import (
	"context"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/openmcp-project/controller-utils/pkg/resources"
	"github.com/openmcp-project/openmcp-operator/api/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"
)

// SecretSync is a helper to sync image pull secrets from the platform cluster to the workload cluster.
// It copies the secrets from the platform cluster namespace to the workload cluster namespace and
// renames them to include the component name as prefix to avoid name clashes. Each copied secret name is guaranteed to be unique
// per component, even if the same image pull secret is used in multiple components.
type SecretSync struct {
	PlatformCluster          *clusters.Cluster
	PlatformClusterNamespace string
	WorkloadCluster          *clusters.Cluster
	WorkloadClusterNamespace string
}

// CreateOrUpdate copies the image pull secrets from the platform cluster to the workload cluster.
// It returns a list of LocalObjectReference that can be used in the PodSpec of the component.
func (s *SecretSync) CreateOrUpdate(ctx context.Context, c *identity.Component, imagePullSecrets []common.LocalObjectReference) ([]corev1.LocalObjectReference, error) {
	imagePullSecretRefs := make([]corev1.LocalObjectReference, 0, len(imagePullSecrets))

	for _, ips := range imagePullSecrets {
		sourceSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ips.Name,
				Namespace: s.PlatformClusterNamespace,
			},
		}

		if err := s.PlatformCluster.Client().Get(ctx, client.ObjectKeyFromObject(sourceSecret), sourceSecret); err != nil {
			return nil, err
		}

		imagePullSecretName := c.ImagePullSecretName(ips.Name)

		if err := resources.CreateOrUpdateResource(ctx, s.WorkloadCluster.Client(), newImagePullSecretMutator(imagePullSecretName, s.WorkloadClusterNamespace, sourceSecret.Data, c)); err != nil {
			return nil, err
		}

		imagePullSecretRefs = append(imagePullSecretRefs, corev1.LocalObjectReference{
			Name: imagePullSecretName,
		})
	}

	return imagePullSecretRefs, nil
}

// Delete removes the image pull secrets from the workload cluster that were copied from the platform cluster.
func (s *SecretSync) Delete(ctx context.Context, c *identity.Component, imagePullSecrets []common.LocalObjectReference) error {
	for _, ips := range imagePullSecrets {
		sourceSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ips.Name,
				Namespace: s.PlatformClusterNamespace,
			},
		}

		if err := s.PlatformCluster.Client().Get(ctx, client.ObjectKeyFromObject(sourceSecret), sourceSecret); err != nil {
			return err
		}

		if err := resources.DeleteResource(ctx, s.WorkloadCluster.Client(), newImagePullSecretMutator(ips.Name, s.WorkloadClusterNamespace, sourceSecret.Data, c)); err != nil {
			return err
		}
	}

	return nil
}

func newImagePullSecretMutator(name, namespace string, secretData map[string][]byte, c *identity.Component) resources.Mutator[*corev1.Secret] {
	m := resources.NewSecretMutator(
		name,
		namespace,
		secretData,
		corev1.SecretTypeDockerConfigJson)
	m.MetadataMutator().WithLabels(c.Labels())
	return m
}
