# Installing the Service Provider Landscaper

Each service provider of an MCP landscape is represented by an [MCPServiceProvider resource](#the-mcpserviceprovider-resource). The MCP operator reconciles this resource and starts the service provider. The essential parts of the service provider landscaper are a Job that executes an [init command](#init-command) and a Deployment that executes a [run command](#run-command).


## The MCPServiceProvider Resource

The MCPServiceProvider CRD is defined in (TODO).

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


## Init Command

The `init` command is executed by a Job, and performs the tasks that must be done once for the setup of the service provider landscaper. Currently, this is the creation of the `Landscaper` CRD.

```shell
init \
  --onboarding-cluster <path to kubeconfig file of the onboarding cluster>
```


## Run Command

The `run` command is executed by a Deployment, and starts the controller which reconciles the `Landscaper` resources.

```shell
run \
  --onboarding-cluster <path to kubeconfig file of the onboarding cluster> \
  --workload-cluster <path to kubeconfig file of the workload cluster> \
  --workload-cluster-domain <domain of the workload cluster> \
  --service-provider-resource-path <path to the service provider resource>
```

The `workload-cluster` and `workload-cluster-domain` arguments are temporary. They will be removed when access to the workload cluster is obtained via a cluster request.
