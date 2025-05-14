package app

import (
	"fmt"
	"os"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/openmcp-project/controller-utils/pkg/logging"

	"github.com/spf13/cobra"
)

type RawSharedOptions struct {
}

type SharedOptions struct {
	*RawSharedOptions

	Clusters *Clusters

	Log logging.Logger
}

type Clusters struct {
	Onboarding *clusters.Cluster
	Platform   *clusters.Cluster
	Workload   *clusters.Cluster
}

func (o *SharedOptions) AddPersistentFlags(cmd *cobra.Command) {
	// logging
	logging.InitFlags(cmd.PersistentFlags())
	// clusters
	o.Clusters.Onboarding.RegisterConfigPathFlag(cmd.PersistentFlags())
	o.Clusters.Platform.RegisterConfigPathFlag(cmd.PersistentFlags())
	o.Clusters.Workload.RegisterConfigPathFlag(cmd.PersistentFlags())
}

func (o *SharedOptions) PrintRaw(cmd *cobra.Command) {
	data, err := yaml.Marshal(o.RawSharedOptions)
	if err != nil {
		cmd.Println(fmt.Errorf("error marshalling raw shared options: %w", err).Error())
		return
	}
	cmd.Print(string(data))
}

func (o *SharedOptions) PrintCompleted(cmd *cobra.Command) {
	raw := map[string]any{
		"clusters": map[string]any{
			"onboarding": o.Clusters.Onboarding.APIServerEndpoint(),
			"platform":   o.Clusters.Platform.APIServerEndpoint(),
			"workload":   o.Clusters.Workload.APIServerEndpoint(),
		},
	}
	data, err := yaml.Marshal(raw)
	if err != nil {
		cmd.Println(fmt.Errorf("error marshalling completed shared options: %w", err).Error())
		return
	}
	cmd.Print(string(data))
}

func (o *SharedOptions) Complete() error {
	if err := o.Clusters.Onboarding.InitializeRESTConfig(); err != nil {
		return err
	}

	if err := o.Clusters.Platform.InitializeRESTConfig(); err != nil {
		return err
	}

	if err := o.Clusters.Workload.InitializeRESTConfig(); err != nil {
		return err
	}

	log, err := logging.GetLogger()
	if err != nil {
		return err
	}
	o.Log = log
	ctrl.SetLogger(o.Log.Logr())

	return nil
}

func NewServiceProviderLandscaperCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "service-provider-landscaper",
		Aliases: []string{"landscaper-provider"},
		Short:   "Commands for interacting with the service provider landscaper",
	}
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	so := &SharedOptions{
		RawSharedOptions: &RawSharedOptions{},
		Clusters: &Clusters{
			Onboarding: clusters.New("onboarding"),
			Platform:   clusters.New("platform"),
			Workload:   clusters.New("workload"),
		},
	}
	so.AddPersistentFlags(cmd)
	cmd.AddCommand(NewInitCommand(so))
	cmd.AddCommand(NewRunCommand(so))

	return cmd
}
