package cluster

import (
	"context"
	"fmt"
	"time"

	"github.com/openmcp-project/controller-utils/pkg/logging"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/openmcp-project/controller-utils/pkg/resources"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterRequestMutator struct {
	Purpose string

	ClusterRequest *clustersv1alpha1.ClusterRequest
	Metadata       resources.MetadataMutator
}

func (m *ClusterRequestMutator) String() string {
	return "ClusterRequestMutator"
}

func (m *ClusterRequestMutator) Empty() *clustersv1alpha1.ClusterRequest {
	return m.ClusterRequest
}

func (m *ClusterRequestMutator) Mutate(clusterRequest *clustersv1alpha1.ClusterRequest) error {
	clusterRequest.Spec = clustersv1alpha1.ClusterRequestSpec{
		Purpose: m.Purpose,
	}

	return nil
}

func (m *ClusterRequestMutator) MetadataMutator() resources.MetadataMutator {
	return m.Metadata
}

type AccessRequestMutator struct {
	RequestRef    clustersv1alpha1.NamespacedObjectReference
	Permissions   []clustersv1alpha1.PermissionsRequest
	AccessRequest *clustersv1alpha1.AccessRequest
	Metadata      resources.MetadataMutator
}

func (m *AccessRequestMutator) String() string {
	return "AccessRequestMutator"
}

func (m *AccessRequestMutator) Empty() *clustersv1alpha1.AccessRequest {
	return m.AccessRequest
}

func (m *AccessRequestMutator) MetadataMutator() resources.MetadataMutator {
	return m.Metadata
}

func (m *AccessRequestMutator) Mutate(accessRequest *clustersv1alpha1.AccessRequest) error {
	accessRequest.Spec = clustersv1alpha1.AccessRequestSpec{
		RequestRef:  &m.RequestRef,
		Permissions: m.Permissions,
	}

	return nil
}

type ClusterManager struct {
	platformCluster *clusters.Cluster
	name            string
	namespace       string
}

func NewClusterManager(name, namespace string, platformCluster *clusters.Cluster) *ClusterManager {
	return &ClusterManager{
		platformCluster: platformCluster,
		name:            name,
		namespace:       namespace,
	}
}

func (cm *ClusterManager) CreateOrUpdateCluster(ctx context.Context, clusterID string, scheme *runtime.Scheme, timeout time.Duration, log *logging.Logger) (*clusters.Cluster, error) {
	requestName := fmt.Sprintf("%s-%s", cm.name, clusterID)

	clusterRequest := &clustersv1alpha1.ClusterRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      requestName,
			Namespace: cm.namespace,
		},
	}

	err := resources.CreateOrUpdateResource(ctx, cm.platformCluster.Client(), &ClusterRequestMutator{
		Purpose:        clusterID,
		ClusterRequest: clusterRequest,
		Metadata:       resources.NewMetadataMutator(),
	})
	if err != nil {
		return nil, err
	}

	// wait for clusterAccess.Status.Phase to be Granted or Denied
	err = waitForResourceCondition(ctx, cm.platformCluster.Client(), clusterRequest, func(cr *clustersv1alpha1.ClusterRequest) (bool, error) {
		if log != nil {
			log.Info("Waiting for cluster request", "name", cr.Name, "phase", cr.Status.Phase)
		}
		return cr.Status.Phase.IsGranted() || cr.Status.Phase.IsDenied(), nil
	}, timeout, 10*time.Second)

	if err != nil {
		return nil, fmt.Errorf("failed to wait for cluster request: %w", err)
	}

	if !clusterRequest.Status.Phase.IsGranted() {
		return nil, fmt.Errorf("failed to wait for cluster request: %q, %q", clusterRequest.Status.Reason, clusterRequest.Status.Message)
	}

	accessRequest := &clustersv1alpha1.AccessRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      requestName,
			Namespace: cm.namespace,
		},
	}

	err = resources.CreateOrUpdateResource(ctx, cm.platformCluster.Client(), &AccessRequestMutator{
		RequestRef: clustersv1alpha1.NamespacedObjectReference{
			ObjectReference: clustersv1alpha1.ObjectReference{
				Name: requestName,
			},
			Namespace: cm.namespace,
		},
		Permissions: []clustersv1alpha1.PermissionsRequest{
			{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"*"},
						Resources: []string{"*"},
						Verbs:     []string{"*"},
					},
				},
			},
		},
		AccessRequest: accessRequest,
	})
	if err != nil {
		return nil, err
	}

	// wait for accessRequest.Status.Phase to be Granted or Denied
	err = waitForResourceCondition(ctx, cm.platformCluster.Client(), accessRequest, func(ar *clustersv1alpha1.AccessRequest) (bool, error) {
		if log != nil {
			log.Info("Waiting for access request", "name", ar.Name, "phase", ar.Status.Phase)
		}
		return ar.Status.Phase.IsGranted() || ar.Status.Phase.IsDenied(), nil
	}, timeout, 10*time.Second)

	if err != nil {
		return nil, fmt.Errorf("failed to wait for access request: %w", err)
	}

	if !accessRequest.Status.Phase.IsGranted() {
		return nil, fmt.Errorf("failed to wait for access request: %q, %q", accessRequest.Status.Reason, accessRequest.Status.Message)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      accessRequest.Status.SecretRef.Name,
			Namespace: accessRequest.Status.SecretRef.Namespace,
		},
	}

	err = cm.platformCluster.Client().Get(ctx, client.ObjectKeyFromObject(secret), secret)
	if err != nil {
		return nil, fmt.Errorf("failed to get access secret: %w", err)
	}

	kubeconfigBytes, ok := secret.Data["kubeconfig"]
	if !ok {
		return nil, fmt.Errorf("kubeconfig not found in access secret %s", secret.Name)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create rest config from kubeconfig bytes: %w", err)
	}

	// Create a new cluster with the access secret
	cluster := clusters.New(clusterID).WithRESTConfig(config)
	if err = cluster.InitializeClient(scheme); err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	return cluster, nil
}

func waitForResourceCondition[T client.Object](ctx context.Context, kubeClient client.Client, obj T, conditionFunc func(obj T) (bool, error), timeout time.Duration, interval time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return wait.PollUntilContextTimeout(ctx, interval, timeout, true, func(ctx context.Context) (bool, error) {
		// Fetch the latest version of the resource
		if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			return false, fmt.Errorf("failed to get resource: %w", err)
		}

		// Check if the condition is met
		return conditionFunc(obj)
	})
}
