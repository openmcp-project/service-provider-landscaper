package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/openmcp-project/controller-utils/pkg/logging"
	openmcpconstv1alpha1 "github.com/openmcp-project/openmcp-operator/api/constants"

	"github.com/openmcp-project/openmcp-operator/lib/clusteraccess"

	"github.com/openmcp-project/service-provider-landscaper/api/v1alpha2"

	rbacv1 "k8s.io/api/rbac/v1"

	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	deploymentv1alpha1 "github.com/openmcp-project/openmcp-operator/api/provider/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	providerscheme "github.com/openmcp-project/service-provider-landscaper/api/install"
	controller1 "github.com/openmcp-project/service-provider-landscaper/internal/controller"
)

var setupLog logging.Logger

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
	RawRunOptions

	// fields filled in Complete()
	TLSOpts              []func(*tls.Config)
	WebhookTLSOpts       []func(*tls.Config)
	MetricsServerOptions metricsserver.Options
	MetricsCertWatcher   *certwatcher.CertWatcher
	WebhookCertWatcher   *certwatcher.CertWatcher
}

type RawRunOptions struct {
	// kubebuilder default flags
	MetricsAddr          string `json:"metrics-bind-address"`
	MetricsCertPath      string `json:"metrics-cert-path"`
	MetricsCertName      string `json:"metrics-cert-name"`
	MetricsCertKey       string `json:"metrics-cert-key"`
	WebhookCertPath      string `json:"webhook-cert-path"`
	WebhookCertName      string `json:"webhook-cert-name"`
	WebhookCertKey       string `json:"webhook-cert-key"`
	EnableLeaderElection bool   `json:"leader-elect"`
	ProbeAddr            string `json:"health-probe-bind-address"`
	PprofAddr            string `json:"pprof-bind-address"`
	SecureMetrics        bool   `json:"metrics-secure"`
	EnableHTTP2          bool   `json:"enable-http2"`
}

func (o *RunOptions) AddFlags(cmd *cobra.Command) {
	// kubebuilder default flags
	cmd.Flags().StringVar(&o.MetricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	cmd.Flags().StringVar(&o.ProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	cmd.Flags().StringVar(&o.PprofAddr, "pprof-bind-address", "", "The address the pprof endpoint binds to. Expected format is ':<port>'. Leave empty to disable pprof endpoint.")
	cmd.Flags().BoolVar(&o.EnableLeaderElection, "leader-elect", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	cmd.Flags().BoolVar(&o.SecureMetrics, "metrics-secure", true, "If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	cmd.Flags().StringVar(&o.WebhookCertPath, "webhook-cert-path", "", "The directory that contains the webhook certificate.")
	cmd.Flags().StringVar(&o.WebhookCertName, "webhook-cert-name", "tls.crt", "The name of the webhook certificate file.")
	cmd.Flags().StringVar(&o.WebhookCertKey, "webhook-cert-key", "tls.key", "The name of the webhook key file.")
	cmd.Flags().StringVar(&o.MetricsCertPath, "metrics-cert-path", "", "The directory that contains the metrics server certificate.")
	cmd.Flags().StringVar(&o.MetricsCertName, "metrics-cert-name", "tls.crt", "The name of the metrics server certificate file.")
	cmd.Flags().StringVar(&o.MetricsCertKey, "metrics-cert-key", "tls.key", "The name of the metrics server key file.")
	cmd.Flags().BoolVar(&o.EnableHTTP2, "enable-http2", false, "If set, HTTP/2 will be enabled for the metrics and webhook servers")

}

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

	setupLog = o.Log.WithName("setup")
	ctrl.SetLogger(o.Log.Logr())

	// kubebuilder default stuff

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("Disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !o.EnableHTTP2 {
		o.TLSOpts = append(o.TLSOpts, disableHTTP2)
	}

	// Initial webhook TLS options
	o.WebhookTLSOpts = o.TLSOpts

	if len(o.WebhookCertPath) > 0 {
		setupLog.Info("Initializing webhook certificate watcher using provided certificates", "webhook-cert-path", o.WebhookCertPath, "webhook-cert-name", o.WebhookCertName, "webhook-cert-key", o.WebhookCertKey)

		var err error
		o.WebhookCertWatcher, err = certwatcher.New(
			filepath.Join(o.WebhookCertPath, o.WebhookCertName),
			filepath.Join(o.WebhookCertPath, o.WebhookCertKey),
		)
		if err != nil {
			return fmt.Errorf("failed to initialize webhook certificate watcher: %w", err)
		}

		o.WebhookTLSOpts = append(o.WebhookTLSOpts, func(config *tls.Config) {
			config.GetCertificate = o.WebhookCertWatcher.GetCertificate
		})
	}

	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	o.MetricsServerOptions = metricsserver.Options{
		BindAddress:   o.MetricsAddr,
		SecureServing: o.SecureMetrics,
		TLSOpts:       o.TLSOpts,
	}

	if o.SecureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/metrics/filters#WithAuthenticationAndAuthorization
		o.MetricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	// If the certificate is not specified, controller-runtime will automatically
	// generate self-signed certificates for the metrics server. While convenient for development and testing,
	// this setup is not recommended for production.
	//
	// TODO(user): If you enable certManager, uncomment the following lines:
	// - [METRICS-WITH-CERTS] at config/default/kustomization.yaml to generate and use certificates
	// managed by cert-manager for the metrics server.
	// - [PROMETHEUS-WITH-CERTS] at config/prometheus/kustomization.yaml for TLS certification.
	if len(o.MetricsCertPath) > 0 {
		setupLog.Info("Initializing metrics certificate watcher using provided certificates", "metrics-cert-path", o.MetricsCertPath, "metrics-cert-name", o.MetricsCertName, "metrics-cert-key", o.MetricsCertKey)

		var err error
		o.MetricsCertWatcher, err = certwatcher.New(
			filepath.Join(o.MetricsCertPath, o.MetricsCertName),
			filepath.Join(o.MetricsCertPath, o.MetricsCertKey),
		)
		if err != nil {
			return fmt.Errorf("failed to initialize metrics certificate watcher: %w", err)
		}

		o.MetricsServerOptions.TLSOpts = append(o.MetricsServerOptions.TLSOpts, func(config *tls.Config) {
			config.GetCertificate = o.MetricsCertWatcher.GetCertificate
		})
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
	utilruntime.Must(deploymentv1alpha1.AddToScheme(platformScheme))
	providerscheme.InstallProviderAPIs(platformScheme)

	onboardingScheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(onboardingScheme))
	providerscheme.InstallProviderAPIs(onboardingScheme)

	if err := o.Clusters.Platform.InitializeClient(platformScheme); err != nil {
		return err
	}

	providerSystemNamespace := os.Getenv(openmcpconstv1alpha1.EnvVariablePodNamespace)
	if providerSystemNamespace == "" {
		return fmt.Errorf("environment variable %s is not set", openmcpconstv1alpha1.EnvVariablePodNamespace)
	}

	clusterAccessManager := clusteraccess.NewClusterAccessManager(o.Clusters.Platform.Client(), v1alpha2.LandscaperProviderName, providerSystemNamespace)
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
		Scheme:                  onboardingScheme,
		Metrics:                 o.MetricsServerOptions,
		HealthProbeBindAddress:  o.ProbeAddr,
		PprofBindAddress:        o.PprofAddr,
		LeaderElection:          o.EnableLeaderElection,
		LeaderElectionID:        "github.com/openmcp-project/service-provider-landscaper",
		LeaderElectionNamespace: providerSystemNamespace,
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		LeaderElectionReleaseOnCancel: true,
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
		ProviderName:      o.ProviderName,
		ProviderNamespace: providerSystemNamespace,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller: %w", err)
	}

	if o.MetricsCertWatcher != nil {
		setupLog.Info("Adding metrics certificate watcher to manager")
		if err := mgr.Add(o.MetricsCertWatcher); err != nil {
			return fmt.Errorf("unable to add metrics certificate watcher to manager: %w", err)
		}
	}

	if o.WebhookCertWatcher != nil {
		setupLog.Info("Adding webhook certificate watcher to manager")
		if err := mgr.Add(o.WebhookCertWatcher); err != nil {
			return fmt.Errorf("unable to add webhook certificate watcher to manager: %w", err)
		}
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}

	o.Log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		o.Log.Error(err, "error while running manager")
		os.Exit(1)
	}

	return nil
}
