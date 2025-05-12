package app

import (
	"context"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"

	providerscheme "github.com/openmcp-project/service-provider-landscaper/api/install"

	"sigs.k8s.io/yaml"

	"github.com/spf13/cobra"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
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

type RawRunOptions struct {
	WorkloadClusterDomain string

	// var metricsAddr string
	// var metricsCertPath, metricsCertName, metricsCertKey string
	// var webhookCertPath, webhookCertName, webhookCertKey string
	// var enableLeaderElection bool
	// var probeAddr string
	// var secureMetrics bool
	// var enableHTTP2 bool
	// var tlsOpts []func(*tls.Config)
}

type RunOptions struct {
	*SharedOptions
	RawRunOptions
}

func (o *RunOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.WorkloadClusterDomain, "workload-cluster-domain", "",
		"Domain of the workload cluster.")
}

func (o *RunOptions) PrintRaw(cmd *cobra.Command) {
	data, err := yaml.Marshal(o.RawRunOptions)
	if err != nil {
		cmd.Println(fmt.Errorf("error marshalling raw options: %w", err).Error())
		return
	}
	cmd.Print(string(data))
}

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

func (o *RunOptions) Run(_ context.Context) error {
	o.Log.Info("running service provider landscaper")

	if err := o.Clusters.Onboarding.InitializeClient(providerscheme.InstallProviderAPIs(runtime.NewScheme())); err != nil {
		return err
	}
	if err := o.Clusters.Platform.InitializeClient(providerscheme.InstallProviderAPIs(runtime.NewScheme())); err != nil {
		return err
	}
	workloadClusterScheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(workloadClusterScheme))
	if err := o.Clusters.Workload.InitializeClient(workloadClusterScheme); err != nil {
		return err
	}

	mgrOptions := ctrl.Options{
		Metrics:        metricsserver.Options{BindAddress: "0"},
		LeaderElection: false,
	}

	mgr, err := ctrl.NewManager(o.Clusters.Onboarding.RESTConfig(), mgrOptions)
	if err != nil {
		return fmt.Errorf("unable to setup manager: %w", err)
	}

	if err = mgr.Add(o.Clusters.Platform.Cluster()); err != nil {
		return fmt.Errorf("unable to add platform cluster to manager: %w", err)
	}

	utilruntime.Must(clientgoscheme.AddToScheme(mgr.GetScheme()))
	utilruntime.Must(api.AddToScheme(mgr.GetScheme()))

	if err = (&controller1.LandscaperReconciler{
		OnboardingCluster:     o.Clusters.Onboarding,
		PlatformCluster:       o.Clusters.Platform,
		WorkloadCluster:       o.Clusters.Workload,
		WorkloadClusterDomain: o.WorkloadClusterDomain,
		Scheme:                mgr.GetScheme(),
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
