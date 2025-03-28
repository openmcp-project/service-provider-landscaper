package app

import (
	"context"

	"github.com/spf13/cobra"
)

func NewServiceProviderLandscaperCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "service-provider-landscaper",
		Aliases: []string{"landscaper-provider"},
		Short:   "Commands for interacting with the service provider landscaper",
	}

	cmd.AddCommand(NewInitCommand(ctx))
	cmd.AddCommand(NewRunCommand(ctx))

	return cmd
}
