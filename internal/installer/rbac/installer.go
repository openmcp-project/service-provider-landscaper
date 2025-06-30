package rbac

import (
	"context"
	_ "embed"

	"github.com/openmcp-project/controller-utils/pkg/clusters"

	"github.com/openmcp-project/controller-utils/pkg/resources"
)

// embed the file data/test-kubeconfig.yaml
//
//go:embed data/test-kubeconfig.yaml
var testKubeconfig []byte

type Kubeconfigs struct {
	MCPCluster      []byte
	WorkloadCluster []byte
}

type KubeconfigAccessor func(ctx context.Context, cluster *clusters.Cluster) ([]byte, error)

func defaultKubeconfigAccessorImpl(_ context.Context, cluster *clusters.Cluster) ([]byte, error) {
	return cluster.WriteKubeconfig()
}

func TestKubeconfigAccessorImpl(_ context.Context, _ *clusters.Cluster) ([]byte, error) {
	return testKubeconfig, nil
}

var kubeconfigAccessor KubeconfigAccessor = defaultKubeconfigAccessorImpl

func SetKubeconfigAccessor(accessor KubeconfigAccessor) {
	kubeconfigAccessor = accessor
}

func GetKubeconfigs(ctx context.Context, values *Values) (*Kubeconfigs, error) {
	var err error
	kubeconfigs := &Kubeconfigs{}

	kubeconfigs.MCPCluster, err = kubeconfigAccessor(ctx, values.MCPCluster)
	if err != nil {
		return kubeconfigs, err
	}

	kubeconfigs.WorkloadCluster, err = kubeconfigAccessor(ctx, values.WorkloadCluster)
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
