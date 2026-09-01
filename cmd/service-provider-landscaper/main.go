package main

import (
	"context"
	"fmt"
	"os"

	"github.com/openmcp-project/service-provider-landscaper/cmd/service-provider-landscaper/app"

	"github.com/openmcp-project/controller-utils/pkg/fips"
)

func main() {
	fips.Verify(context.Background())

	cmd := app.NewServiceProviderLandscaperCommand()

	if err := cmd.Execute(); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}
