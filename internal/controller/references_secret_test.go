package controller

import (
	"github.com/openmcp-project/openmcp-operator/api/common"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("referencesSecret", func() {
	It("should return false for empty refs", func() {
		Expect(referencesSecret(nil, "any")).To(BeFalse())
		Expect(referencesSecret([]common.LocalObjectReference{}, "any")).To(BeFalse())
	})

	It("should return true when the secret name matches", func() {
		refs := []common.LocalObjectReference{{Name: "my-secret"}}
		Expect(referencesSecret(refs, "my-secret")).To(BeTrue())
	})

	It("should return false when the secret name does not match", func() {
		refs := []common.LocalObjectReference{{Name: "other-secret"}}
		Expect(referencesSecret(refs, "my-secret")).To(BeFalse())
	})

	It("should find a match among multiple refs", func() {
		refs := []common.LocalObjectReference{
			{Name: "first"},
			{Name: "target"},
			{Name: "last"},
		}
		Expect(referencesSecret(refs, "target")).To(BeTrue())
	})

	It("should return false when no ref matches among multiple", func() {
		refs := []common.LocalObjectReference{
			{Name: "first"},
			{Name: "second"},
		}
		Expect(referencesSecret(refs, "missing")).To(BeFalse())
	})
})
