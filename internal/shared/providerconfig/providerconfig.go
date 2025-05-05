package providerconfig

import (
	"fmt"
	"os"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
)

type serviceProviderSpec struct {
	ProviderConfig api.LandscaperProviderConfiguration `json:"providerConfig"`
}

type serviceProvider struct {
	Spec serviceProviderSpec `json:"spec"`
}

// ReadProviderConfig reads a ServiceProvider yaml file and returns the landscaper provider specific config.
func ReadProviderConfig(serviceProviderResourcePath string) (*api.LandscaperProviderConfiguration, error) {
	if serviceProviderResourcePath == "" {
		return nil, fmt.Errorf("service provider resource path is required")
	}

	data, err := os.ReadFile(serviceProviderResourcePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read service provider resource file: %w", err)
	}

	p := &serviceProvider{}
	err = yaml.Unmarshal(data, p)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal service provider resource: %w", err)
	}

	return &p.Spec.ProviderConfig, nil
}

// ReadProviderConfigFromSecret reads a ServiceProvider yaml file and returns the landscaper provider specific config.
func ReadProviderConfigFromSecret(serviceProviderResourcePath string) (*api.LandscaperProviderConfiguration, error) {
	if serviceProviderResourcePath == "" {
		return nil, fmt.Errorf("service provider resource path is required")
	}

	data, err := os.ReadFile(serviceProviderResourcePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read service provider resource file: %w", err)
	}

	s := &v1.Secret{}
	err = yaml.Unmarshal(data, s)

	providerBytes, found := s.Data["mcpServiceProvider"]
	if !found {
		return nil, fmt.Errorf("unable to find service provider resource: %w", err)
	}

	p := &serviceProvider{}
	err = yaml.Unmarshal(providerBytes, p)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal service provider resource: %w", err)
	}

	return &p.Spec.ProviderConfig, nil
}
