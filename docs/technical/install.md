# Installing the Service Provider Landscaper

Each service provider of an MCP landscape is represented by an `MCPServiceProvider` resource. 

Section `spec.deploymentConfig` has the same structure for all service providers. It contains the configuration for the deployment of the service provider.

Section `spec.providerConfig` contains the provider specific configuration. Its structure is provider specific. For the service provider landscaper, it contains the images needed for the landscaper instances (landscaper controller, the landscaper webhooks server, the helm deployer, and the manifest deployer).

```yaml
apiVersion: openmcp.cloud/v1alpha1
kind: MCPServiceProvider
metadata:
  name: service-provider-landscaper
spec:

  # configuration for the deployment of the service provider  
  deploymentConfig:
    oci:
      image: "service-provider-landscaper:v0.0.1"
      pullSecrets: []
    deployment: {}

  # provider specific configuration    
  providerConfig:
    landscaperController:
      image: "...landscaper-controller:v0.127.0"
    landscaperWebhooksServer:
      image: "...landscaper-webhooks-server:v0.127.0"
    helmDeployer:
      image: "...helm-deployer-controller:v0.127.0"
    manifestDeployer:
      image: "...manifest-deployer-controller:v0.127.0"
```
