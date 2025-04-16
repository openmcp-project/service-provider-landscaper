package cluster

import (
	"fmt"
	"os"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
)

// WorkloadCluster will be replaced when the cluster scheduler is available.
func WorkloadCluster() (*clusters.Cluster, error) {
	workloadCluster := clusters.New("workload").WithConfigPath(os.Getenv("WORKLOAD_KUBECONFIG_PATH"))
	if err := workloadCluster.InitializeRESTConfig(); err != nil {
		return nil, fmt.Errorf("failed to initialize rest config for workload cluster: %w", err)
	}
	if err := workloadCluster.InitializeClient(nil); err != nil {
		return nil, fmt.Errorf("failed to initialize rest config for workload cluster: %w", err)
	}
	return workloadCluster, nil
}

// MCPCluster will be replaced when the cluster scheduler is available.
func MCPCluster() (*clusters.Cluster, error) {
	mcpCluster := clusters.New("mcp").WithConfigPath(os.Getenv("MCP_KUBECONFIG_PATH"))
	if err := mcpCluster.InitializeRESTConfig(); err != nil {
		return nil, fmt.Errorf("failed to initialize rest config for mcp cluster: %w", err)
	}
	if err := mcpCluster.InitializeClient(nil); err != nil {
		return nil, fmt.Errorf("failed to initialize rest config for mcp cluster: %w", err)
	}
	return mcpCluster, nil
}
