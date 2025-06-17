package rbac

import (
	"context"

	"github.com/openmcp-project/controller-utils/pkg/resources"

	"github.com/openmcp-project/service-provider-landscaper/internal/shared/cluster"
)

type Kubeconfigs struct {
	ControllerKubeconfig []byte
	WebhooksKubeconfig   []byte
}

func InstallLandscaperRBACResources(ctx context.Context, values *Values) (kubeconfigs *Kubeconfigs, err error) {
	valHelper, err := newValuesHelper(values)
	if err != nil {
		return kubeconfigs, err
	}

	mcpClient := values.MCPCluster.Client()

	if err := resources.CreateOrUpdateResource(ctx, mcpClient, resources.NewNamespaceMutator(valHelper.resourceNamespace())); err != nil {
		return kubeconfigs, err
	}

	// Create kubeconfigs for the MCP cluster
	kubeconfigs = &Kubeconfigs{}

	kubeconfigs.ControllerKubeconfig, err = cluster.CreateKubeconfig(values.MCPCluster)
	if err != nil {
		return kubeconfigs, err
	}

	kubeconfigs.WebhooksKubeconfig, err = cluster.CreateKubeconfig(values.MCPCluster)
	if err != nil {
		return kubeconfigs, err
	}

	return kubeconfigs, nil
}

func UninstallLandscaperRBACResources(ctx context.Context, values *Values) error {

	valHelper, err := newValuesHelper(values)
	if err != nil {
		return err
	}

	mcpClient := values.MCPCluster.Client()

	if err := resources.DeleteResource(ctx, mcpClient, resources.NewNamespaceMutator(valHelper.resourceNamespace())); err != nil {
		return err
	}

	return nil
}
