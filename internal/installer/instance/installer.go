package instance

import (
	"context"
	"fmt"

	"github.com/openmcp-project/controller-utils/pkg/readiness"

	"github.com/openmcp-project/service-provider-landscaper/internal/installer/helmdeployer"
	"github.com/openmcp-project/service-provider-landscaper/internal/installer/landscaper"
	"github.com/openmcp-project/service-provider-landscaper/internal/installer/manifestdeployer"
	"github.com/openmcp-project/service-provider-landscaper/internal/installer/rbac"
)

func InstallLandscaperInstance(ctx context.Context, config *Configuration) error {
	kubeconfigs, err := rbac.GetKubeconfigs(ctx, rbacValues(config))
	if err != nil {
		return fmt.Errorf("failed to get kubeconfigs: %w", err)
	}

	// RBAC resources
	err = rbac.InstallLandscaperRBACResources(ctx, rbacValues(config))
	if err != nil {
		return fmt.Errorf("failed to install landscaper rbac resources: %v", err)
	}

	// Manifest deployer
	manifestExports, err := manifestdeployer.InstallManifestDeployer(ctx, manifestDeployerValues(config, kubeconfigs))
	if err != nil {
		return fmt.Errorf("failed to install manifest deployer: %w", err)
	}

	// Helm deployer
	helmExports, err := helmdeployer.InstallHelmDeployer(ctx, helmDeployerValues(config, kubeconfigs))
	if err != nil {
		return fmt.Errorf("failed to install helm deployer: %w", err)
	}

	// Landscaper
	err = landscaper.InstallLandscaper(ctx, landscaperValues(config, kubeconfigs, manifestExports, helmExports))
	if err != nil {
		return fmt.Errorf("failed to install landscaper controllers: %w", err)
	}

	return nil
}

func UninstallLandscaperInstance(ctx context.Context, config *Configuration) error {
	kubeconfigs, err := rbac.GetKubeconfigs(ctx, rbacValues(config))
	if err != nil {
		return fmt.Errorf("failed to get kubeconfigs: %w", err)
	}

	err = landscaper.UninstallLandscaper(ctx, landscaperValues(config, kubeconfigs, nil, nil))
	if err != nil {
		return fmt.Errorf("failed to uninstall landscaper controllers: %w", err)
	}

	err = helmdeployer.UninstallHelmDeployer(ctx, helmDeployerValues(config, kubeconfigs))
	if err != nil {
		return fmt.Errorf("failed to uninstall helm deployer: %w", err)
	}

	err = manifestdeployer.UninstallManifestDeployer(ctx, manifestDeployerValues(config, kubeconfigs))
	if err != nil {
		return fmt.Errorf("failed to uninstall manifest deployer: %w", err)
	}

	err = rbac.UninstallLandscaperRBACResources(ctx, rbacValues(config))
	if err != nil {
		return fmt.Errorf("failed to uninstall landscaper rbac resources: %v", err)
	}
	return nil
}

func CheckReadiness(ctx context.Context, config *Configuration) readiness.CheckResult {
	kubeconfigs := &rbac.Kubeconfigs{}
	return readiness.Aggregate(
		manifestdeployer.CheckReadiness(ctx, manifestDeployerValues(config, kubeconfigs)),
		helmdeployer.CheckReadiness(ctx, helmDeployerValues(config, kubeconfigs)),
		landscaper.CheckReadiness(ctx, landscaperValues(config, kubeconfigs, nil, nil)),
	)
}
