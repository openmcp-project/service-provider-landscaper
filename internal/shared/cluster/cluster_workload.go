package cluster

import (
	"fmt"
	"os"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
)

// WorkloadCluster creates a cluster object representing a workload cluster.
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
