package cluster

import (
	"fmt"

	"github.com/openmcp-project/controller-utils/pkg/clusters"

	"k8s.io/client-go/tools/clientcmd"
	clientapi "k8s.io/client-go/tools/clientcmd/api"
)

func CreateKubeconfig(cluster *clusters.Cluster) ([]byte, error) {
	var contextName string
	var kubeconfigCluster *clientapi.Cluster
	var authInfo *clientapi.AuthInfo

	if cluster.HasID() {
		contextName = cluster.ID()
	} else {
		contextName = "default"
	}

	if cluster.RESTConfig() == nil {
		// create a fake kubeconfig for testing
		kubeconfigCluster = &clientapi.Cluster{
			Server:                   "https://fake-server",
			CertificateAuthorityData: []byte("fake-ca-data"),
		}
		authInfo = &clientapi.AuthInfo{
			Token: "fake-token",
		}
	} else {
		// use the actual cluster's REST config
		kubeconfigCluster = &clientapi.Cluster{
			Server:                   cluster.RESTConfig().Host,
			CertificateAuthorityData: cluster.RESTConfig().CAData,
		}
		authInfo = &clientapi.AuthInfo{
			Token: cluster.RESTConfig().BearerToken,
		}

	}

	kubeConfig := clientapi.Config{
		CurrentContext: contextName,
		Contexts: map[string]*clientapi.Context{
			contextName: {
				AuthInfo: contextName,
				Cluster:  contextName,
			},
		},
		Clusters: map[string]*clientapi.Cluster{
			contextName: kubeconfigCluster,
		},
		AuthInfos: map[string]*clientapi.AuthInfo{
			contextName: authInfo,
		},
	}

	kubeconfigYaml, err := clientcmd.Write(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kubeconfig: %w", err)
	}

	return kubeconfigYaml, nil
}
