package cluster

import (
	"fmt"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Cluster interface {
	Client() client.Client
	RESTConfig() *rest.Config
}

var _ Cluster = (*clusters.Cluster)(nil)
var _ Cluster = (*clusterImpl)(nil)

func newClusterFromKubeconfigBytes(kubeconfigBytes []byte, scheme *runtime.Scheme) (Cluster, error) {
	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create rest config from kubeconfig: %w", err)
	}

	cl, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create client from rest config: %w", err)
	}

	return &clusterImpl{
		restCfg: restCfg,
		client:  cl,
	}, nil
}

type clusterImpl struct {
	restCfg *rest.Config
	client  client.Client
}

func (c *clusterImpl) Client() client.Client {
	return c.client
}

func (c *clusterImpl) RESTConfig() *rest.Config {
	return c.restCfg
}
