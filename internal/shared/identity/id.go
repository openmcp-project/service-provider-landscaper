package identity

import (
	"crypto/sha1"
	"encoding/base32"
	"fmt"

	"github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
)

const (
	labelInstanceID          = "landscaper.services.openmcp.cloud/instance-id"
	base32EncodeStdLowerCase = "abcdefghijklmnopqrstuvwxyz234567"
)

func GetInstanceID(ls *v1alpha1.Landscaper) string {
	return ls.Labels[labelInstanceID]
}

func SetInstanceID(ls *v1alpha1.Landscaper, tenantID string) {
	if ls.Labels == nil {
		ls.Labels = map[string]string{}
	}
	ls.Labels[labelInstanceID] = tenantID
}

func ComputeInstanceID(ls *v1alpha1.Landscaper) string {
	// TODO: use utils.K8sNameHash of the mcp-operator
	h := sha1.New()
	_, _ = fmt.Fprintf(h, ls.Namespace, "/", ls.Name)
	id := base32.NewEncoding(base32EncodeStdLowerCase).WithPadding(base32.NoPadding).EncodeToString(h.Sum(nil))
	return id
}
