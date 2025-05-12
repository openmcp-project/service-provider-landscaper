package main

import (
	"fmt"
	"os"

	"github.com/openmcp-project/service-provider-landscaper/cmd/service-provider-landscaper/app"
)

func main() {
	cmd := app.NewServiceProviderLandscaperCommand()

	if err := cmd.Execute(); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}
