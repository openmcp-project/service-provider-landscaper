package landscaper

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/openmcp-project/controller-utils/pkg/readiness"
	"github.com/openmcp-project/controller-utils/pkg/resources"
)

func InstallLandscaper(ctx context.Context, values *Values) error {

	valHelper, err := newValuesHelper(values)
	if err != nil {
		return err
	}

	workloadClient := values.WorkloadCluster.Client()

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, resources.NewNamespaceMutator(valHelper.workloadNamespace())); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, newControllerMCPKubeconfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, newControllerWorkloadKubeconfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, newWebhooksKubeconfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, newConfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, newServiceMutator(valHelper)); err != nil {
		return err
	}

	if !valHelper.areAllWebhooksDisabled() {
		if err := resources.CreateOrUpdateResource(ctx, workloadClient, newWebhooksServiceMutator(valHelper)); err != nil {
			return err
		}
	}

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, newCentralDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, newMainDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if !valHelper.areAllWebhooksDisabled() {
		if err := resources.CreateOrUpdateResource(ctx, workloadClient, newWebhooksDeploymentMutator(valHelper)); err != nil {
			return err
		}
	}

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, newMainHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, newCentralHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, workloadClient, newWebhooksHPAMutator(valHelper)); err != nil {
		return err
	}

	return nil
}

func UninstallLandscaper(ctx context.Context, values *Values) error {

	valHelper, err := newValuesHelperForDelete(values)
	if err != nil {
		return err
	}

	workloadClient := values.WorkloadCluster.Client()

	if err := resources.DeleteResource(ctx, workloadClient, newWebhooksHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, workloadClient, newCentralHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, workloadClient, newMainHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, workloadClient, newWebhooksDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, workloadClient, newMainDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, workloadClient, newCentralDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, workloadClient, newWebhooksServiceMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, workloadClient, newServiceMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, workloadClient, newConfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, workloadClient, newWebhooksKubeconfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, workloadClient, newControllerMCPKubeconfigSecretMutator(valHelper)); err != nil {
		return err
	}
	if err := resources.DeleteResource(ctx, workloadClient, newControllerWorkloadKubeconfigSecretMutator(valHelper)); err != nil {
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

	aggregatedResult := readiness.NewReadyResult()
	for _, mut := range []resources.Mutator[*appsv1.Deployment]{
		newCentralDeploymentMutator(valHelper),
		newMainDeploymentMutator(valHelper),
		newWebhooksDeploymentMutator(valHelper),
	} {
		dp, err := resources.GetResource(ctx, hostClient, mut)
		if err != nil {
			return readiness.NewFailedResult(err)
		}
		result := readiness.CheckDeployment(dp)
		aggregatedResult = readiness.Aggregate(aggregatedResult, result)
	}

	return aggregatedResult
}
