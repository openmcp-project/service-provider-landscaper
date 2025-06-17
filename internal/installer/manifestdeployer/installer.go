package manifestdeployer

import (
	"context"

	"github.com/openmcp-project/controller-utils/pkg/readiness"
	"github.com/openmcp-project/controller-utils/pkg/resources"
)

type Exports struct {
	DeploymentName string
}

func InstallManifestDeployer(ctx context.Context, values *Values) (*Exports, error) {

	valHelper, err := newValuesHelper(values)
	if err != nil {
		return nil, err
	}

	hostClient := values.WorkloadCluster.Client()

	if err := resources.CreateOrUpdateResource(ctx, hostClient, resources.NewNamespaceMutator(valHelper.workloadNamespace())); err != nil {
		return nil, err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newConfigSecretMutator(valHelper)); err != nil {
		return nil, err
	}

	if len(valHelper.mcpClusterKubeconfig()) > 0 {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newKubeconfigSecretMutator(valHelper)); err != nil {
			return nil, err
		}
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newHPAMutator(valHelper)); err != nil {
		return nil, err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newDeploymentMutator(valHelper)); err != nil {
		return nil, err
	}

	return &Exports{
		// needed for health checks
		DeploymentName: valHelper.manifestDeployerComponent.NamespacedDefaultResourceName(),
	}, nil
}

func UninstallManifestDeployer(ctx context.Context, values *Values) error {

	valHelper, err := newValuesHelperForDelete(values)
	if err != nil {
		return err
	}

	hostClient := values.WorkloadCluster.Client()

	if err := resources.DeleteResource(ctx, hostClient, newDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newKubeconfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newConfigSecretMutator(valHelper)); err != nil {
		return err
	}

	return nil
}

func CheckReadiness(ctx context.Context, values *Values) readiness.CheckResult {
	valHelper, err := newValuesHelperForDelete(values)
	if err != nil {
		return readiness.NewFailedResult(err)
	}

	hostClient := values.WorkloadCluster.Client()
	dp, err := resources.GetResource(ctx, hostClient, newDeploymentMutator(valHelper))
	if err != nil {
		return readiness.NewFailedResult(err)
	}
	return readiness.CheckDeployment(dp)
}
