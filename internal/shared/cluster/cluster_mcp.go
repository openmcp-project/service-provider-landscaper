package cluster

import (
	"context"
	"fmt"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MCPCluster creates a cluster object representing an MCP cluster.
// Temporarily, we assume that the kubeconfig comes from a secret in the onboarding cluster.
// We assume that mcp and landscaper object have the same object key, and that the kubeconfig of the mcp cluster
// is stored in a secret with the name <mcp name>.kubeconfig.
func MCPCluster(ctx context.Context, lsObjectKey client.ObjectKey, onboardingClient client.Client) (*clusters.Cluster, error) {
	k := client.ObjectKey{Namespace: lsObjectKey.Namespace, Name: lsObjectKey.Name + ".kubeconfig"}
	s := &core.Secret{}
	if err := onboardingClient.Get(ctx, k, s); err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", k, err)
	}
	kubeconfigBytes, ok := s.Data["kubeconfig"]
	if !ok {
		return nil, fmt.Errorf("kubeconfig not found in secret %s", k)
	}
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create rest config from kubeconfig bytes: %w", err)
	}
	mcp := clusters.New("mcp").WithRESTConfig(config)
	scheme := runtime.NewScheme()
	if err = clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add client-go scheme: %w", err)
	}

	if err = mcp.InitializeClient(scheme); err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}
	return mcp, nil
}
