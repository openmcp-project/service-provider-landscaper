package providerconfig

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
)

// ReadProviderConfig reads a ServiceProvider yaml file and returns the landscaper provider specific config.
func ReadProviderConfig(serviceProviderResourcePath string) (*api.ProviderConfig, error) {
	if serviceProviderResourcePath == "" {
		return nil, fmt.Errorf("service provider resource path is required")
	}

	data, err := os.ReadFile(serviceProviderResourcePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read service provider resource file: %w", err)
	}

	p := &api.ProviderConfig{}
	err = yaml.Unmarshal(data, p)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal service provider resource: %w", err)
	}

	return p, nil
}

// ReadProviderConfigFromSecret reads a ServiceProvider yaml file and returns the landscaper provider specific config.
func ReadProviderConfigFromSecret(serviceProviderResourcePath string) (*api.ProviderConfig, error) {
	if serviceProviderResourcePath == "" {
		return nil, fmt.Errorf("service provider resource path is required")
	}

	data, err := os.ReadFile(serviceProviderResourcePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read service provider resource file: %w", err)
	}

	p := &api.ProviderConfig{}
	err = yaml.Unmarshal(data, p)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal service provider resource: %w", err)
	}

	return p, nil
}
