package app

import (
	"context"
	goflag "flag"
	"fmt"
	"os"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/openmcp-project/controller-utils/pkg/logging"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/openmcp-project/service-provider-landscaper/api/install"
	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
	controller1 "github.com/openmcp-project/service-provider-landscaper/internal/controller"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/providerconfig"
)

func NewRunCommand(_ context.Context) *cobra.Command {
	options := &runOptions{
		OnboardingCluster: clusters.New("onboarding"),
		WorkloadCluster:   clusters.New("workload"),
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start the service provider landscaper",
		Run: func(cmd *cobra.Command, args []string) {
			if err := options.complete(); err != nil {
				fmt.Print(err)
				os.Exit(1)
			}
			if err := options.run(); err != nil {
				options.Log.Error(err, "unable to run service provider landscaper")
				os.Exit(1)
			}
		},
	}

	options.addFlags(cmd.Flags())

	return cmd
}

type runOptions struct {
	OnboardingCluster           *clusters.Cluster
	WorkloadCluster             *clusters.Cluster
	WorkloadClusterDomain       string
	ServiceProviderResourcePath string

	Log                   logging.Logger
	ServiceProviderConfig *api.LandscaperProviderConfiguration

	// var metricsAddr string
	// var metricsCertPath, metricsCertName, metricsCertKey string
	// var webhookCertPath, webhookCertName, webhookCertKey string
	// var enableLeaderElection bool
	// var probeAddr string
	// var secureMetrics bool
	// var enableHTTP2 bool
	// var tlsOpts []func(*tls.Config)
}

func (o *runOptions) addFlags(fs *flag.FlagSet) {
	// register flag '--onboarding-cluster' for the path to the kubeconfig of the onboarding cluster
	o.OnboardingCluster.RegisterConfigPathFlag(fs)

	// register flag '--workload-cluster' for the path to the kubeconfig of the workload cluster
	o.WorkloadCluster.RegisterConfigPathFlag(fs)

	fs.StringVar(&o.WorkloadClusterDomain, "workload-cluster-domain", "",
		"Domain of the workload cluster.")

	fs.StringVar(&o.ServiceProviderResourcePath, "service-provider-resource-path", "",
		"Path to the yaml manifest of the service provider landscaper.")

	logging.InitFlags(fs)
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

func (o *runOptions) complete() (err error) {
	if err = o.setupLogger(); err != nil {
		return err
	}
	if err = o.setupOnboardingClusterClient(); err != nil {
		return err
	}
	if err = o.setupWorkloadClusterClient(); err != nil {
		return err
	}

	o.ServiceProviderConfig, err = providerconfig.ReadProviderConfig(o.ServiceProviderResourcePath)
	if err != nil {
		return err
	}

	return nil
}

func (o *runOptions) setupLogger() error {
	log, err := logging.GetLogger()
	if err != nil {
		return err
	}
	o.Log = log
	ctrl.SetLogger(log.Logr())
	return nil
}

func (o *runOptions) setupOnboardingClusterClient() error {
	if err := o.OnboardingCluster.InitializeRESTConfig(); err != nil {
		return fmt.Errorf("unable to initialize onboarding cluster rest config: %w", err)
	}
	if err := o.OnboardingCluster.InitializeClient(install.InstallCRDAPIs(runtime.NewScheme())); err != nil {
		return fmt.Errorf("unable to initialize onboarding cluster client: %w", err)
	}
	return nil
}

func (o *runOptions) setupWorkloadClusterClient() error {
	if err := o.WorkloadCluster.InitializeRESTConfig(); err != nil {
		return fmt.Errorf("unable to initialize workload cluster rest config: %w", err)
	}
	if err := o.WorkloadCluster.InitializeClient(install.InstallCRDAPIs(runtime.NewScheme())); err != nil {
		return fmt.Errorf("unable to initialize workload cluster client: %w", err)
	}
	return nil
}

func (o *runOptions) run() error {
	o.Log.Info("starting service provider landscaper",
		"onboarding-cluster", o.OnboardingCluster.ConfigPath(),
		"service-provider-resource-path", o.ServiceProviderResourcePath)

	mgrOptions := ctrl.Options{
		Metrics:        metricsserver.Options{BindAddress: "0"},
		LeaderElection: false,
	}

	mgr, err := ctrl.NewManager(o.OnboardingCluster.RESTConfig(), mgrOptions)
	if err != nil {
		return fmt.Errorf("unable to setup manager: %w", err)
	}

	utilruntime.Must(clientgoscheme.AddToScheme(mgr.GetScheme()))
	utilruntime.Must(api.AddToScheme(mgr.GetScheme()))

	if err = (&controller1.LandscaperReconciler{
		OnboardingClient:         mgr.GetClient(),
		WorkloadCluster:          o.WorkloadCluster,
		WorkloadClusterDomain:    o.WorkloadClusterDomain,
		Scheme:                   mgr.GetScheme(),
		LandscaperProviderConfig: o.ServiceProviderConfig,
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
