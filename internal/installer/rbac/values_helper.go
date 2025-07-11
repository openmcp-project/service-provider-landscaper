package rbac

import (
	"fmt"

	"github.com/openmcp-project/service-provider-landscaper/internal/shared/identity"
)

const (
	componentLandscaperRBAC = "landscaper-rbac"
)

type valuesHelper struct {
	values        *Values
	rbacComponent *identity.Component
}

func newValuesHelper(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}

	return &valuesHelper{
		values:        values,
		rbacComponent: identity.NewComponent(values.Instance, values.Version, componentLandscaperRBAC),
	}, nil
}

func (h *valuesHelper) resourceNamespace() string {
	return h.values.Instance.Namespace()
}
