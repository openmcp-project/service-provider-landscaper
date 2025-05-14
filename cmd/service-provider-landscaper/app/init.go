package app

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	ctrlutil "github.com/openmcp-project/controller-utils/pkg/controller"
	"github.com/openmcp-project/controller-utils/pkg/resources"
	"github.com/spf13/cobra"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"

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

	if err := o.Clusters.Onboarding.InitializeClient(providerscheme.InstallCRDAPIs(runtime.NewScheme())); err != nil {
		return err
	}
	if err := o.Clusters.Platform.InitializeClient(providerscheme.InstallCRDAPIs(runtime.NewScheme())); err != nil {
		return err
	}

	if err := o.createOrUpdateCRDs(ctx); err != nil {
		return err
	}

	o.Log.Info("finished init command")
	return nil
}

func (o *InitOptions) createOrUpdateCRDs(ctx context.Context) error {
	crdList := crds.CRDs()
	var errs error
	for _, crd := range crdList {
		c, err := o.clusterForCRD(crd)
		if err != nil {
			return err
		}

		o.Log.Info("creating/updating CRD", "name", crd.Name, "cluster", c.ID())
		err = resources.CreateOrUpdateResource(ctx, c.Client(), resources.NewCRDMutator(crd, nil, nil))
		errs = errors.Join(errs, err)
	}
	if errs != nil {
		return fmt.Errorf("error creating/updating CRDs: %w", errs)
	}
	return nil
}

func (o *InitOptions) clusterForCRD(crd *apiextv1.CustomResourceDefinition) (*clusters.Cluster, error) {
	purpose, _ := ctrlutil.GetLabel(crd, clustersv1alpha1.ClusterLabel)
	switch purpose {
	case clustersv1alpha1.PURPOSE_ONBOARDING:
		return o.Clusters.Onboarding, nil
	case clustersv1alpha1.PURPOSE_PLATFORM:
		return o.Clusters.Platform, nil
	default:
		return nil, fmt.Errorf("missing cluster label '%s' or unsupported value '%s' for CRD '%s'",
			clustersv1alpha1.ClusterLabel, purpose, crd.Name)
	}
}
