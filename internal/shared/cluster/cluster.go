package cluster

import (
	"fmt"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Cluster struct {
	kubeconfig []byte
	restConfig *rest.Config
	client     client.Client
}

func NewCluster(kubeconfigPath string) (*Cluster, error) {

	kubeconfig, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read kubeconfig file: %w", err)
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create kubernetes rest config: %w", err)
	}

	client, err := client.New(restConfig, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("unable to create kubernetes client: %w", err)
	}

	return &Cluster{
		kubeconfig: kubeconfig,
		restConfig: restConfig,
		client:     client,
	}, nil
}

func (c *Cluster) Kubeconfig() []byte {
	return c.kubeconfig
}

func (c *Cluster) RestConfig() *rest.Config {
	return c.restConfig
}

func (c *Cluster) Client() client.Client {
	return c.client
}
