package app

import (
	"context"
	"fmt"
	"os"
	"time"

	openmcpconstv1alpha1 "github.com/openmcp-project/openmcp-operator/api/constants"

	"github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"

	rbacv1 "k8s.io/api/rbac/v1"

	crdutil "github.com/openmcp-project/controller-utils/pkg/crds"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/openmcp-project/openmcp-operator/lib/clusteraccess"

	"github.com/openmcp-project/service-provider-landscaper/api/crds"
	providerscheme "github.com/openmcp-project/service-provider-landscaper/api/install"
)

func NewInitCommand(so *SharedOptions) *cobra.Command {
	opts := &InitOptions{
		SharedOptions: so,
	}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the service provider landscaper",
		Run: func(cmd *cobra.Command, args []string) {
			opts.PrintRawOptions(cmd)
			if err := opts.Complete(cmd.Context()); err != nil {
				panic(fmt.Errorf("error completing options: %w", err))
			}
			opts.PrintCompletedOptions(cmd)
			if err := opts.Run(cmd.Context()); err != nil {
				panic(err)
			}
		},
	}

	opts.AddFlags(cmd)

	return cmd
}

type InitOptions struct {
	*SharedOptions
}

func (o *InitOptions) AddFlags(cmd *cobra.Command) {}

func (o *InitOptions) PrintRaw(cmd *cobra.Command) {}

func (o *InitOptions) PrintRawOptions(cmd *cobra.Command) {
	cmd.Println("########## RAW OPTIONS START ##########")
	o.SharedOptions.PrintRaw(cmd)
	o.PrintRaw(cmd)
	cmd.Println("########## RAW OPTIONS END ##########")
}

func (o *InitOptions) Complete(ctx context.Context) error {
	if err := o.SharedOptions.Complete(); err != nil {
		return err
	}
	return nil
}

func (o *InitOptions) PrintCompleted(cmd *cobra.Command) {}

func (o *InitOptions) PrintCompletedOptions(cmd *cobra.Command) {
	cmd.Println("########## COMPLETED OPTIONS START ##########")
	o.SharedOptions.PrintCompleted(cmd)
	o.PrintCompleted(cmd)
	cmd.Println("########## COMPLETED OPTIONS END ##########")
}

func (o *InitOptions) Run(ctx context.Context) error {
	o.Log.Info("initializing service provider landscaper")

	platformScheme := runtime.NewScheme()
	providerscheme.InstallCRDAPIs(platformScheme)
	utilruntime.Must(clustersv1alpha1.AddToScheme(platformScheme))

	onboardingScheme := runtime.NewScheme()
	providerscheme.InstallCRDAPIs(onboardingScheme)

	if err := o.Clusters.Platform.InitializeClient(platformScheme); err != nil {
		return err
	}

	providerSystemNamespace := os.Getenv(openmcpconstv1alpha1.EnvVariablePodNamespace)
	if providerSystemNamespace == "" {
		return fmt.Errorf("environment variable %s is not set", openmcpconstv1alpha1.EnvVariablePodNamespace)
	}

	clusterAccessManager := clusteraccess.NewClusterAccessManager(o.Clusters.Platform.Client(), v1alpha1.LandscaperProviderName, providerSystemNamespace)
	clusterAccessManager.WithLogger(&o.Log).
		WithInterval(10 * time.Second).
		WithTimeout(30 * time.Minute)

	onboardingCluster, err := clusterAccessManager.CreateAndWaitForCluster(ctx, "onboarding", clustersv1alpha1.PURPOSE_ONBOARDING,
		onboardingScheme, []clustersv1alpha1.PermissionsRequest{
			{
				// TODO: define the specific permissions needed for the onboarding cluster
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"*"},
						Resources: []string{"*"},
						Verbs:     []string{"*"},
					},
				},
			},
		})

	if err != nil {
		return fmt.Errorf("error creating/updating onboarding cluster: %w", err)
	}

	crdManager := crdutil.NewCRDManager(openmcpconstv1alpha1.ClusterLabel, crds.CRDs)

	crdManager.AddCRDLabelToClusterMapping(clustersv1alpha1.PURPOSE_PLATFORM, o.Clusters.Platform)
	crdManager.AddCRDLabelToClusterMapping(clustersv1alpha1.PURPOSE_ONBOARDING, onboardingCluster)

	if err := crdManager.CreateOrUpdateCRDs(ctx, &o.Log); err != nil {
		return fmt.Errorf("error creating/updating CRDs: %w", err)
	}

	o.Log.Info("finished init command")
	return nil
}
