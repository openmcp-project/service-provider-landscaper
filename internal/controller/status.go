package controller

import (
	"github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newCreateOrUpdateStatus(generation int64) *v1alpha1.LandscaperStatus {
	s := &v1alpha1.LandscaperStatus{
		ObservedGeneration: generation,
		Phase:              v1alpha1.Progressing,
	}

	apimeta.SetStatusCondition(&s.Conditions, meta.Condition{
		Type:               "Installed",
		Status:             meta.ConditionUnknown,
		ObservedGeneration: generation,
		Reason:             "InstallationPending",
	})

	apimeta.SetStatusCondition(&s.Conditions, meta.Condition{
		Type:               "Ready",
		Status:             meta.ConditionUnknown,
		ObservedGeneration: generation,
		Reason:             "ReadinessCheckPending",
	})

}
