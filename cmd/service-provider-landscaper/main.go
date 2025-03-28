package main

import (
	"context"
	"fmt"
	"os"

	"github.com/openmcp-project/service-provider-landscaper/cmd/service-provider-landscaper/app"
)

func main() {
	ctx := context.Background()
	defer ctx.Done()
	cmd := app.NewServiceProviderLandscaperCommand(ctx)

	if err := cmd.Execute(); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}
