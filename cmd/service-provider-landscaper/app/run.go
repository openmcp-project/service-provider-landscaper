package app

import (
	"context"
	goflag "flag"
	"fmt"
	"os"

	"github.com/openmcp-project/controller-utils/pkg/logging"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
	controller1 "github.com/openmcp-project/service-provider-landscaper/internal/controller"
	"github.com/openmcp-project/service-provider-landscaper/internal/shared/providerconfig"
)

func NewRunCommand(ctx context.Context) *cobra.Command {
	options := &runOptions{}

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
	OnboardingKubeconfigPath    string
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
	fs.StringVar(&o.OnboardingKubeconfigPath, "onboarding-kubeconfig-path", "",
		"Path to the kubeconfig of the onboarding cluster.")
	fs.StringVar(&o.ServiceProviderResourcePath, "service-provider-resource-path", "",
		"Path to the yaml manifest of the service provider landscaper.")

	logging.InitFlags(fs)
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

func (o *runOptions) complete() (err error) {
	if err = o.setupLogger(); err != nil {
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

func (o *runOptions) run() error {
	o.Log.Info("Starting service provider landscaper",
		"onboarding-kubeconfig-path", o.OnboardingKubeconfigPath,
		"service-provider-resource-path", o.ServiceProviderResourcePath)

	onboardingKubeconfig, err := os.ReadFile(o.OnboardingKubeconfigPath)
	if err != nil {
		o.Log.Error(err, "unable to read onboarding cluster kubeconfig")
		os.Exit(1)
	}
	onboardingRestConfig, err := clientcmd.RESTConfigFromKubeConfig(onboardingKubeconfig)
	if err != nil {
		o.Log.Error(err, "unable to create onboarding cluster rest config")
		os.Exit(1)
	}

	mgrOptions := ctrl.Options{
		Metrics:        metricsserver.Options{BindAddress: "0"},
		LeaderElection: false,
	}

	mgr, err := ctrl.NewManager(onboardingRestConfig, mgrOptions)
	if err != nil {
		return fmt.Errorf("unable to setup manager: %w", err)
	}

	utilruntime.Must(clientgoscheme.AddToScheme(mgr.GetScheme()))
	utilruntime.Must(api.AddToScheme(mgr.GetScheme()))

	if err = (&controller1.LandscaperReconciler{
		OnboardingClient:         mgr.GetClient(),
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
