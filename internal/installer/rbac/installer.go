package rbac

import (
	"context"

	"github.com/openmcp-project/controller-utils/pkg/resources"

	"github.com/openmcp-project/service-provider-landscaper/internal/shared/cluster"
)

type Kubeconfigs struct {
	MCPCluster      []byte
	WorkloadCluster []byte
}

func GetKubeconfigs(ctx context.Context, values *Values) (*Kubeconfigs, error) {
	var err error
	kubeconfigs := &Kubeconfigs{}

	kubeconfigs.MCPCluster, err = cluster.CreateKubeconfig(values.MCPCluster)
	if err != nil {
		return kubeconfigs, err
	}

	kubeconfigs.WorkloadCluster, err = cluster.CreateKubeconfig(values.WorkloadCluster)
	if err != nil {
		return kubeconfigs, err
	}

	return kubeconfigs, nil
}

func InstallLandscaperRBACResources(ctx context.Context, values *Values) error {
	valHelper, err := newValuesHelper(values)
	if err != nil {
		return err
	}

	mcpClient := values.MCPCluster.Client()

	if err = resources.CreateOrUpdateResource(ctx, mcpClient, resources.NewNamespaceMutator(valHelper.resourceNamespace())); err != nil {
		return err
	}

	return nil
}

func UninstallLandscaperRBACResources(ctx context.Context, values *Values) error {
	valHelper, err := newValuesHelper(values)
	if err != nil {
		return err
	}

	mcpClient := values.MCPCluster.Client()

	if err = resources.DeleteResource(ctx, mcpClient, resources.NewNamespaceMutator(valHelper.resourceNamespace())); err != nil {
		return err
	}

	return nil
}
