package controller

import (
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
)

func initConditions(ls *api.Landscaper) {
	apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
		Type:               "Installed",
		Status:             meta.ConditionUnknown,
		ObservedGeneration: ls.Generation,
		Reason:             "InstallationPending",
	})
	apimeta.SetStatusCondition(&ls.Status.Conditions, meta.Condition{
		Type:               "Ready",
		Status:             meta.ConditionUnknown,
		ObservedGeneration: ls.Generation,
		Reason:             "ReadinessCheckPending",
	})
}
