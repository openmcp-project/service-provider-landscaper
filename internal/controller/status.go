package controller

import (
	"github.com/openmcp-project/controller-utils/pkg/readiness"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"
)

const (
	typeInstalled   = "Installed"
	typeUninstalled = "Uninstalled"
	typeReady       = "Ready"

	reasonInstallationPending    = "InstallationPending"
	reasonReadinessCheckPending  = "ReadinessCheckPending"
	reasonWaitForLandscaperReady = "WaitForLandscaperReady"
	reasonUninstallationPending  = "UninstallationPending"

	reasonLandscaperInstalled = "LandscaperInstalled"
	reasonLandscaperReady     = "LandscaperReady"

	reasonInstallFailed       = "InstallFailed"
	reasonClusterAccessError  = "ClusterAccessError"
	reasonProviderConfigError = "ProviderConfigError"
	reasonConfigurationError  = "ConfigurationError"

	messageWaitingForClusterAccessReady = "Waiting for cluster access to be ready"
)

type reconcileStatus struct {
	InstallCondition   *meta.Condition
	ReadyCondition     *meta.Condition
	UninstallCondition *meta.Condition
	ObservedGeneration int64
	Phase              v1alpha1.LandscaperPhase
}

func (s *reconcileStatus) setInstallWaitForClusterAccessReady() {
	s.InstallCondition = &meta.Condition{
		Type:               typeInstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             reasonInstallationPending,
		Message:            messageWaitingForClusterAccessReady,
	}
}

func (s *reconcileStatus) setUninstallWaitForClusterAccessReady() {
	s.InstallCondition = &meta.Condition{
		Type:               typeInstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             reasonInstallationPending,
		Message:            messageWaitingForClusterAccessReady,
	}
}

func (s *reconcileStatus) setInstalled() {
	s.InstallCondition = &meta.Condition{
		Type:               typeInstalled,
		Status:             meta.ConditionTrue,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             reasonLandscaperInstalled,
		Message:            "Landscaper has been installed successfully",
	}
}

func (s *reconcileStatus) setUninstalled() {
	s.UninstallCondition = &meta.Condition{
		Type:               typeUninstalled,
		Status:             meta.ConditionTrue,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             reasonLandscaperInstalled,
		Message:            "Landscaper has been uninstalled successfully",
	}
}

func (s *reconcileStatus) setWaitForReadinessCheck(result readiness.CheckResult) {
	s.ReadyCondition = &meta.Condition{
		Type:               typeReady,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             reasonWaitForLandscaperReady,
		Message:            result.Message(),
	}
}

func (s *reconcileStatus) setReady() {
	s.ReadyCondition = &meta.Condition{
		Type:               typeReady,
		Status:             meta.ConditionTrue,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             reasonLandscaperReady,
		Message:            "Landscaper is ready",
	}

	s.Phase = v1alpha1.Ready
}

func (s *reconcileStatus) setInstallFailed(err error) {
	s.InstallCondition = &meta.Condition{
		Type:               typeInstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             reasonInstallFailed,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setUninstallFailed(err error) {
	s.UninstallCondition = &meta.Condition{
		Type:               typeUninstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             reasonInstallFailed,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setInstallClusterAccessError(err error) {
	s.InstallCondition = &meta.Condition{
		Type:               typeInstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             reasonClusterAccessError,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setUninstallClusterAccessError(err error) {
	s.UninstallCondition = &meta.Condition{
		Type:               typeUninstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             reasonClusterAccessError,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setInstallProviderConfigError(err error) {
	s.InstallCondition = &meta.Condition{
		Type:               typeInstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             reasonProviderConfigError,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setUninstallProviderConfigError(err error) {
	s.UninstallCondition = &meta.Condition{
		Type:               typeUninstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             reasonProviderConfigError,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setInstallConfigurationError(err error) {
	s.InstallCondition = &meta.Condition{
		Type:               typeInstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             reasonConfigurationError,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setUninstallConfigurationError(err error) {
	s.InstallCondition = &meta.Condition{
		Type:               typeUninstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             reasonConfigurationError,
		Message:            err.Error(),
	}
}

func newCreateOrUpdateStatus(generation int64) *reconcileStatus {
	s := &reconcileStatus{
		ObservedGeneration: generation,
		Phase:              v1alpha1.Progressing,
	}

	s.InstallCondition = &meta.Condition{
		Type:               typeInstalled,
		Status:             meta.ConditionUnknown,
		ObservedGeneration: generation,
		Reason:             reasonInstallationPending,
	}

	s.ReadyCondition = &meta.Condition{
		Type:               typeReady,
		Status:             meta.ConditionUnknown,
		ObservedGeneration: generation,
		Reason:             reasonReadinessCheckPending,
	}

	return s
}

func newDeleteStatus(generation int64) *reconcileStatus {
	s := &reconcileStatus{
		ObservedGeneration: generation,
		Phase:              v1alpha1.Terminating,
	}

	s.UninstallCondition = &meta.Condition{
		Type:               typeUninstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: generation,
		Reason:             reasonUninstallationPending,
	}

	return s
}

func (s *reconcileStatus) convertToLandscaperStatus(status *v1alpha1.LandscaperStatus) {
	status.ObservedGeneration = s.ObservedGeneration
	status.Phase = s.Phase

	if s.InstallCondition != nil {
		apimeta.SetStatusCondition(&status.Conditions, *s.InstallCondition)
	} else {
		apimeta.RemoveStatusCondition(&status.Conditions, typeInstalled)
	}

	if s.ReadyCondition != nil {
		apimeta.SetStatusCondition(&status.Conditions, *s.ReadyCondition)
	} else {
		apimeta.RemoveStatusCondition(&status.Conditions, typeReady)
	}

	if s.UninstallCondition != nil {
		apimeta.SetStatusCondition(&status.Conditions, *s.UninstallCondition)
	} else {
		apimeta.RemoveStatusCondition(&status.Conditions, typeUninstalled)
	}
}
