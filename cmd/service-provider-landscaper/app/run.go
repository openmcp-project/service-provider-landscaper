package app

import (
	"context"
	"fmt"
	"os"
	"time"

	openmcpconstv1alpha1 "github.com/openmcp-project/openmcp-operator/api/constants"

	"github.com/openmcp-project/openmcp-operator/lib/clusteraccess"

	"github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"

	rbacv1 "k8s.io/api/rbac/v1"

	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	providerscheme "github.com/openmcp-project/service-provider-landscaper/api/install"
	controller1 "github.com/openmcp-project/service-provider-landscaper/internal/controller"
)

func NewRunCommand(so *SharedOptions) *cobra.Command {
	opts := &RunOptions{
		SharedOptions: so,
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start the service provider landscaper",
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

type RunOptions struct {
	*SharedOptions
}

func (o *RunOptions) AddFlags(cmd *cobra.Command) {}

func (o *RunOptions) PrintRaw(cmd *cobra.Command) {}

func (o *RunOptions) PrintRawOptions(cmd *cobra.Command) {
	cmd.Println("########## RAW OPTIONS START ##########")
	o.SharedOptions.PrintRaw(cmd)
	o.PrintRaw(cmd)
	cmd.Println("########## RAW OPTIONS END ##########")
}

func (o *RunOptions) Complete(_ context.Context) (err error) {
	if err := o.SharedOptions.Complete(); err != nil {
		return err
	}

	return nil
}

func (o *RunOptions) PrintCompleted(cmd *cobra.Command) {}

func (o *RunOptions) PrintCompletedOptions(cmd *cobra.Command) {
	cmd.Println("########## COMPLETED OPTIONS START ##########")
	o.SharedOptions.PrintCompleted(cmd)
	o.PrintCompleted(cmd)
	cmd.Println("########## COMPLETED OPTIONS END ##########")
}

func (o *RunOptions) Run(ctx context.Context) error {
	o.Log.Info("running service provider landscaper")

	platformScheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(platformScheme))
	utilruntime.Must(clustersv1alpha1.AddToScheme(platformScheme))
	providerscheme.InstallProviderAPIs(platformScheme)

	onboardingScheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(onboardingScheme))
	providerscheme.InstallProviderAPIs(onboardingScheme)

	if err := o.Clusters.Platform.InitializeClient(platformScheme); err != nil {
		return err
	}

	providerSystemNamespace := os.Getenv(openmcpconstv1alpha1.EnvVariablePlatformClusterNamespace)
	if providerSystemNamespace == "" {
		return fmt.Errorf("environment variable %s is not set", openmcpconstv1alpha1.EnvVariablePlatformClusterNamespace)
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

	mgrOptions := ctrl.Options{
		Metrics:        metricsserver.Options{BindAddress: "0"},
		LeaderElection: false,
		Scheme:         onboardingScheme,
	}

	mgr, err := ctrl.NewManager(onboardingCluster.RESTConfig(), mgrOptions)
	if err != nil {
		return fmt.Errorf("unable to setup manager: %w", err)
	}

	if err = mgr.Add(o.Clusters.Platform.Cluster()); err != nil {
		return fmt.Errorf("unable to add platform cluster to manager: %w", err)
	}

	if err = (&controller1.LandscaperReconciler{
		OnboardingCluster: onboardingCluster,
		PlatformCluster:   o.Clusters.Platform,
		Scheme:            mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller: %w", err)
	}

	o.Log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		o.Log.Error(err, "error while running manager")
		os.Exit(1)
	}

	return nil
}
