package cluster

import (
	"context"
	"fmt"

	"github.com/openmcp-project/controller-utils/pkg/clusters"

	auth "k8s.io/api/authentication/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	clientapi "k8s.io/client-go/tools/clientcmd/api"
)

func CreateKubeconfig(ctx context.Context, cluster *clusters.Cluster, serviceAccount *core.ServiceAccount) ([]byte, error) {
	token, err := requestToken(ctx, cluster, serviceAccount)
	if err != nil {
		return nil, fmt.Errorf("failed to request token for service account %s/%s: %w", serviceAccount.Namespace, serviceAccount.Name, err)
	}

	contextName := fmt.Sprintf("%s-%s", serviceAccount.Namespace, serviceAccount.Name)

	var kubeconfigCluster *clientapi.Cluster

	if cluster.RESTConfig() == nil {
		// create a fake kubeconfig for testing
		kubeconfigCluster = &clientapi.Cluster{
			Server:                   "https://fake-server",
			CertificateAuthorityData: []byte("fake-ca-data"),
		}
	} else {
		// use the actual cluster's REST config
		kubeconfigCluster = &clientapi.Cluster{
			Server:                   cluster.RESTConfig().Host,
			CertificateAuthorityData: cluster.RESTConfig().CAData,
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
			contextName: {
				Token: token,
			},
		},
	}

	kubeconfigYaml, err := yaml.Marshal(&kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kubeconfig: %w", err)
	}

	return kubeconfigYaml, nil
}

func requestToken(ctx context.Context, cluster *clusters.Cluster, serviceAccount *core.ServiceAccount) (string, error) {

	tokenRequest := &auth.TokenRequest{
		Spec: auth.TokenRequestSpec{
			ExpirationSeconds: ptr.To[int64](7776000),
		},
	}

	if err := cluster.Client().SubResource("token").Create(ctx, serviceAccount, tokenRequest); err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	return tokenRequest.Status.Token, nil
}
