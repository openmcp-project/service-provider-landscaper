package helmdeployer

import (
	"context"

	"github.com/openmcp-project/controller-utils/pkg/readiness"
	"github.com/openmcp-project/controller-utils/pkg/resources"

	imgpullsecrets "github.com/openmcp-project/service-provider-landscaper/internal/shared/imagepullsecrets"
)

type Exports struct {
	DeploymentName string
}

func InstallHelmDeployer(ctx context.Context, values *Values) (*Exports, error) {

	valHelper, err := newValuesHelper(values)
	if err != nil {
		return nil, err
	}

	workloadClient := values.WorkloadCluster.Client()

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, resources.NewNamespaceMutator(valHelper.workloadNamespace())); err != nil {
		return nil, err
	}

	imgPullSecretsSync := imgpullsecrets.SecretSync{
		PlatformCluster:          values.PlatformCluster,
		PlatformClusterNamespace: values.PlatformClusterNamespace,
		WorkloadCluster:          values.WorkloadCluster,
		WorkloadClusterNamespace: valHelper.workloadNamespace(),
	}

	imagePullSecrets, err := imgPullSecretsSync.CreateOrUpdate(ctx, valHelper.helmDeployerComponent, values.Image.ImagePullSecrets)
	if err != nil {
		return nil, err
	}

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, newConfigSecretMutator(valHelper)); err != nil {
		return nil, err
	}

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, newKubeconfigSecretMutator(valHelper)); err != nil {
		return nil, err
	}

	if valHelper.values.OCI != nil {
		if err := resources.CreateOrUpdateResource(ctx, workloadClient, newRegistrySecretMutator(valHelper)); err != nil {
			return nil, err
		}
	}

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, newHPAMutator(valHelper)); err != nil {
		return nil, err
	}

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, newDeploymentMutator(valHelper).WithImagePullSecrets(imagePullSecrets).Convert()); err != nil {
		return nil, err
	}

	return &Exports{
		// needed for health checks
		DeploymentName: valHelper.helmDeployerComponent.NamespacedDefaultResourceName(),
	}, nil
}

func UninstallHelmDeployer(ctx context.Context, values *Values) error {

	valHelper, err := newValuesHelperForDelete(values)
	if err != nil {
		return err
	}

	workloadClient := values.WorkloadCluster.Client()

	if err := resources.DeleteResource(ctx, workloadClient, newDeploymentMutator(valHelper).Convert()); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, workloadClient, newHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, workloadClient, newKubeconfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, workloadClient, newConfigSecretMutator(valHelper)); err != nil {
		return err
	}

	imgPullSecretsSync := imgpullsecrets.SecretSync{
		PlatformCluster:          values.PlatformCluster,
		PlatformClusterNamespace: values.PlatformClusterNamespace,
		WorkloadCluster:          values.WorkloadCluster,
		WorkloadClusterNamespace: valHelper.workloadNamespace(),
	}

	if err := imgPullSecretsSync.Delete(ctx, valHelper.helmDeployerComponent, values.Image.ImagePullSecrets); err != nil {
		return nil
	}

	return nil
}

func CheckReadiness(ctx context.Context, values *Values) readiness.CheckResult {
	valHelper, err := newValuesHelperForDelete(values)
	if err != nil {
		return readiness.NewFailedResult(err)
	}

	hostClient := values.WorkloadCluster.Client()
	dp, err := resources.GetResource(ctx, hostClient, newDeploymentMutator(valHelper).Convert())
	if err != nil {
		return readiness.NewFailedResult(err)
	}
	return readiness.CheckDeployment(dp)
}
