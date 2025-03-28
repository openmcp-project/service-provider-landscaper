package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func NewInitCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the service provider landscaper",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Initializing the service provider landscaper")
		},
	}

	return cmd
}
