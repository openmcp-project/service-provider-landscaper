package containerdeployer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"

	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"
)

const (
	componentContainerDeployer = "container-deployer"
)

type valuesHelper struct {
	values *Values

	containerDeployerComponent *identity.Component

	configYaml          []byte
	configHash          string
	registrySecretsYaml []byte
	registrySecretsHash string
	registrySecretsData map[string][]byte
}

func newValuesHelper(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}
	if err := values.Default(); err != nil {
		return nil, fmt.Errorf("failed to apply default container deployer values: %w", err)
	}

	// compute values
	configYaml, err := yaml.Marshal(values.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal container deployer config: %w", err)
	}
	configHash := sha256.Sum256(configYaml)

	registrySecretsYaml, err := yaml.Marshal(values.OCI)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal helm deployer config: %w", err)
	}
	registrySecretsHash := sha256.Sum256(registrySecretsYaml)

	registrySecretsData := make(map[string][]byte)
	if values.OCI != nil {
		for key, valueObj := range values.OCI.Secrets {
			valueBytes, err := json.Marshal(valueObj)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal value for registries secret: %w", err)
			}
			registrySecretsData[key] = valueBytes
		}
	}

	return &valuesHelper{
		values:                     values,
		containerDeployerComponent: identity.NewComponent(values.Instance, values.Version, componentContainerDeployer),
		configYaml:                 configYaml,
		configHash:                 hex.EncodeToString(configHash[:]),
		registrySecretsYaml:        registrySecretsYaml,
		registrySecretsHash:        hex.EncodeToString(registrySecretsHash[:]),
		registrySecretsData:        registrySecretsData,
	}, nil
}

func newValuesHelperForDelete(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}
	if err := values.Default(); err != nil {
		return nil, fmt.Errorf("failed to apply default container deployer values during delete operation: %w", err)
	}

	return &valuesHelper{
		values:                     values,
		containerDeployerComponent: identity.NewComponent(values.Instance, values.Version, componentContainerDeployer),
	}, nil
}

func (h *valuesHelper) workloadNamespace() string {
	return h.values.Instance.Namespace()
}

func (h *valuesHelper) mcpKubeconfigSecretName() string {
	return h.containerDeployerComponent.NamespacedResourceName("mcp-kubeconfig")
}

func (h *valuesHelper) mcpClusterKubeconfig() []byte {
	return []byte(h.values.MCPClusterKubeconfig)
}
