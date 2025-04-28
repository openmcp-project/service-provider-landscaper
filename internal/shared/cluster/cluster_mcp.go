package cluster

import (
	"context"
	"fmt"

	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MCPCluster creates a cluster object representing an MCP cluster.
// Temporarily, we assume that the kubeconfig comes from a secret in the onboarding cluster.
// We assume that mcp and landscaper object have the same object key, and that the kubeconfig of the mcp cluster
// is stored in a secret with the name <mcp name>.kubeconfig.
func MCPCluster(ctx context.Context, lsObjectKey client.ObjectKey, onboardingClient client.Client) (Cluster, error) {
	k := client.ObjectKey{Namespace: lsObjectKey.Namespace, Name: lsObjectKey.Name + ".kubeconfig"}
	s := &core.Secret{}
	if err := onboardingClient.Get(ctx, k, s); err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", k, err)
	}
	kubeconfigBytes, ok := s.Data["kubeconfig"]
	if !ok {
		return nil, fmt.Errorf("kubeconfig not found in secret %s", k)
	}
	return newClusterFromKubeconfigBytes(kubeconfigBytes, nil)
}
