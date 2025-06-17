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

	hostClient := values.WorkloadCluster.Client()

	if err := resources.CreateOrUpdateResource(ctx, hostClient, resources.NewNamespaceMutator(valHelper.workloadNamespace())); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newControllerMCPKubeconfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newControllerWorkloadKubeconfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newWebhooksKubeconfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newConfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newServiceMutator(valHelper)); err != nil {
		return err
	}

	if !valHelper.areAllWebhooksDisabled() {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newWebhooksServiceMutator(valHelper)); err != nil {
			return err
		}
	}

	if valHelper.values.WebhooksServer.Ingress != nil {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newIngressMutator(valHelper)); err != nil {
			return err
		}
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newCentralDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newMainDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if !valHelper.areAllWebhooksDisabled() {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newWebhooksDeploymentMutator(valHelper)); err != nil {
			return err
		}
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newMainHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newCentralHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newWebhooksHPAMutator(valHelper)); err != nil {
		return err
	}

	return nil
}

func UninstallLandscaper(ctx context.Context, values *Values) error {

	valHelper, err := newValuesHelperForDelete(values)
	if err != nil {
		return err
	}

	hostClient := values.WorkloadCluster.Client()

	if err := resources.DeleteResource(ctx, hostClient, newWebhooksHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newCentralHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newMainHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newWebhooksDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newMainDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newCentralDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newIngressMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newWebhooksServiceMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newServiceMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newConfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newWebhooksKubeconfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newControllerMCPKubeconfigSecretMutator(valHelper)); err != nil {
		return err
	}
	if err := resources.DeleteResource(ctx, hostClient, newControllerWorkloadKubeconfigSecretMutator(valHelper)); err != nil {
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
