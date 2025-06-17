package helmdeployer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"

	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"
)

const (
	componentHelmDeployer = "helm-deployer"
)

type valuesHelper struct {
	values *Values

	helmDeployerComponent *identity.Component

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
		return nil, fmt.Errorf("failed to apply default helm deployer values: %w", err)
	}

	// compute values
	configYaml, err := yaml.Marshal(values.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal helm deployer config: %w", err)
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
		values:                values,
		helmDeployerComponent: identity.NewComponent(values.Instance, values.Version, componentHelmDeployer),
		configYaml:            configYaml,
		configHash:            hex.EncodeToString(configHash[:]),
		registrySecretsYaml:   registrySecretsYaml,
		registrySecretsHash:   hex.EncodeToString(registrySecretsHash[:]),
		registrySecretsData:   registrySecretsData,
	}, nil
}

func newValuesHelperForDelete(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}
	if err := values.Default(); err != nil {
		return nil, fmt.Errorf("failed to apply default helm deployer values during delete operation: %w", err)
	}

	return &valuesHelper{
		values:                values,
		helmDeployerComponent: identity.NewComponent(values.Instance, values.Version, componentHelmDeployer),
	}, nil
}

func (h *valuesHelper) hostNamespace() string {
	return h.values.Instance.Namespace()
}

func (h *valuesHelper) landscaperClusterKubeconfig() []byte {
	return []byte(h.values.MCPClusterKubeconfig.Kubeconfig)
}

func (h *valuesHelper) isCreateServiceAccount() bool {
	return h.values.ServiceAccount != nil && h.values.ServiceAccount.Create
}
