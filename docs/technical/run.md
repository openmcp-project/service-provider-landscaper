# Running/Debugging the Service Provider Landscaper

## Init Command

```shell
init \
  --onboarding-cluster <path to kubeconfig file of the onboarding cluster>
```


## Run Command

Create an `MCPServiceProvider` resource with name `service-provider-landscaper` (cluster-scoped).

Start the landscaper service provider with the `run` command:

```shell
run \
  --onboarding-cluster <path to kubeconfig file of the onboarding cluster> \
  --workload-cluster <path to kubeconfig file of the workload cluster> \
  --workload-cluster-domain <domain of the workload cluster> \
  --service-provider-resource-path <path to the service provider resource>
```

The `workload-cluster` and `workload-cluster-domain` arguments are temporary. They will be removed when access to the workload cluster is obtained via a cluster request.


## Create Landscaper Instance

In the intended scenario, the user has to create an MCP resource with some name <name> and namespace <namespace>. Temporarily, we skip the MCP resource, and assume that the kubeconfig for the MCP cluster can be obtained from a Secret with name `<name>.kubeconfig` in namespace `<namespace>`.

```shell
kubectl create secret generic "${NAME}.kubeconfig" -n "${NAMESPACE}" \
  --from-file=kubeconfig=<path to kubeconfig file of the mcp cluster>
```
Next the user has to create a Landscaper resource with the name <name> and namespace <namespace>.

```shell

```yaml
apiVersion: landscaper.services.openmcp.cloud/v1alpha1
kind: Landscaper
metadata:
  name: ${NAME}
  namespace: ${NAMESPACE}
spec: {}
```
