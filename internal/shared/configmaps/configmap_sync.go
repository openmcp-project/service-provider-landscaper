package configmapsync

import (
	"context"
	"errors"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/openmcp-project/controller-utils/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CustomCaPath       = "/etc/custom-ca"
	CustomCaVolumeName = "custom-ca-bundle"
	SystemCaPath       = "/etc/ssl/certs"
)

var ErrNilSourceConfigMapRef = errors.New("caBundleRef must not be nil")

// ConfigMapSync is a helper to sync configmaps from the platform cluster to the workload cluster.
// It copies a selected key from the platform cluster namespace to the workload cluster namespace and
// renames the copied configmap to avoid name clashes between components.
type ConfigMapSync struct {
	PlatformCluster          *clusters.Cluster
	PlatformClusterNamespace string
	WorkloadCluster          *clusters.Cluster
	WorkloadClusterNamespace string
}

func (s *ConfigMapSync) CreateOrUpdate(ctx context.Context, caBundleRef *corev1.ConfigMapKeySelector) (*corev1.ConfigMapKeySelector, error) {
	if caBundleRef == nil {
		return nil, ErrNilSourceConfigMapRef
	}

	sourceCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caBundleRef.Name,
			Namespace: s.PlatformClusterNamespace,
		},
	}

	if err := s.PlatformCluster.Client().Get(ctx, client.ObjectKeyFromObject(sourceCM), sourceCM); err != nil {
		return nil, err
	}

	cmName := caBundleRef.Name

	if err := resources.CreateOrUpdateResource(ctx, s.WorkloadCluster.Client(), newCAConfigMapMutator(cmName, s.WorkloadClusterNamespace, sourceCM.Data)); err != nil {
		return nil, err
	}

	return &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: cmName,
		},
		Key: caBundleRef.Key,
	}, nil
}

func (s *ConfigMapSync) Delete(ctx context.Context, caBundleRef *corev1.ConfigMapKeySelector) error {
	if caBundleRef == nil {
		return ErrNilSourceConfigMapRef
	}

	sourceCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caBundleRef.Name,
			Namespace: s.WorkloadClusterNamespace,
		},
	}

	if err := s.WorkloadCluster.Client().Get(ctx, client.ObjectKeyFromObject(sourceCM), sourceCM); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, s.WorkloadCluster.Client(), newCAConfigMapMutator(caBundleRef.Name, s.WorkloadClusterNamespace, sourceCM.Data)); err != nil {
		return err
	}
	return nil
}

func newCAConfigMapMutator(name, namespace string, data map[string]string) resources.Mutator[*corev1.ConfigMap] {
	m := resources.NewConfigMapMutator(
		name,
		namespace,
		data,
	)
	return m
}
