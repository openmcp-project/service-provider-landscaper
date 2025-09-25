package controller

import (
	"github.com/openmcp-project/controller-utils/pkg/readiness"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openmcp-project/service-provider-landscaper/api/v1alpha2"
)

const (
	messageWaitingForClusterAccessReady = "Waiting for cluster access to be ready"
)

type reconcileStatus struct {
	InstallCondition   *meta.Condition
	ReadyCondition     *meta.Condition
	UninstallCondition *meta.Condition
	ObservedGeneration int64
	Phase              v1alpha2.LandscaperPhase
}

func (s *reconcileStatus) setInstallWaitForClusterAccessReady() {
	s.InstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeInstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonInstallationPending,
		Message:            messageWaitingForClusterAccessReady,
	}
}

func (s *reconcileStatus) setUninstallWaitForClusterAccessReady() {
	s.UninstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeUninstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonInstallationPending,
		Message:            messageWaitingForClusterAccessReady,
	}
}

func (s *reconcileStatus) setInstalled() {
	s.InstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeInstalled,
		Status:             meta.ConditionTrue,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonLandscaperInstalled,
		Message:            "Landscaper has been installed successfully",
	}
}

func (s *reconcileStatus) setUninstalled() {
	s.UninstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeUninstalled,
		Status:             meta.ConditionTrue,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonLandscaperInstalled,
		Message:            "Landscaper has been uninstalled successfully",
	}
}

func (s *reconcileStatus) setWaitForReadinessCheck(result readiness.CheckResult) {
	s.ReadyCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeReady,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonWaitForLandscaperReady,
		Message:            result.Message(),
	}
}

func (s *reconcileStatus) setReady() {
	s.ReadyCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeReady,
		Status:             meta.ConditionTrue,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonLandscaperReady,
		Message:            "Landscaper is ready",
	}

	s.Phase = v1alpha2.PhaseReady
}

func (s *reconcileStatus) setInstallFailed(err error) {
	s.InstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeInstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonInstallFailed,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setUninstallFailed(err error) {
	s.UninstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeUninstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonInstallFailed,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setInstallClusterAccessError(err error) {
	s.InstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeInstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonClusterAccessError,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setUninstallClusterAccessError(err error) {
	s.UninstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeUninstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonClusterAccessError,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setInstallProviderConfigError(err error) {
	s.InstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeInstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonProviderConfigError,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setUninstallProviderConfigError(err error) {
	s.UninstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeUninstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonProviderConfigError,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setInstallConfigurationError(err error) {
	s.InstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeInstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonConfigurationError,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setUninstallConfigurationError(err error) {
	s.UninstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeUninstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonConfigurationError,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setInstallDNSConfigFailed(err error) {
	s.InstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeInstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonDNSConfigFailed,
		Message:            err.Error(),
	}
}

func (s *reconcileStatus) setInstallWaitForDNSReady() {
	s.InstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeInstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonWaitForDNSReady,
		Message:            "Waiting for DNS to be ready",
	}
}

func (s *reconcileStatus) SetUninstallDNSConfigFailed(err error) {
	s.UninstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeUninstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: s.ObservedGeneration,
		Reason:             v1alpha2.ConditionReasonDNSConfigFailed,
		Message:            err.Error(),
	}
}

func newCreateOrUpdateStatus(generation int64) *reconcileStatus {
	s := &reconcileStatus{
		ObservedGeneration: generation,
		Phase:              v1alpha2.PhaseProgressing,
	}

	s.InstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeInstalled,
		Status:             meta.ConditionUnknown,
		ObservedGeneration: generation,
		Reason:             v1alpha2.ConditionReasonInstallationPending,
	}

	s.ReadyCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeReady,
		Status:             meta.ConditionUnknown,
		ObservedGeneration: generation,
		Reason:             v1alpha2.ConditionReasonReadinessCheckPending,
	}

	return s
}

func newDeleteStatus(generation int64) *reconcileStatus {
	s := &reconcileStatus{
		ObservedGeneration: generation,
		Phase:              v1alpha2.PhaseTerminating,
	}

	s.UninstallCondition = &meta.Condition{
		Type:               v1alpha2.ConditionTypeUninstalled,
		Status:             meta.ConditionFalse,
		ObservedGeneration: generation,
		Reason:             v1alpha2.ConditionReasonUninstallationPending,
	}

	return s
}

func (s *reconcileStatus) convertToLandscaperStatus(status *v1alpha2.LandscaperStatus) {
	status.ObservedGeneration = s.ObservedGeneration
	status.Phase = s.Phase

	if s.InstallCondition != nil {
		apimeta.SetStatusCondition(&status.Conditions, *s.InstallCondition)
	} else {
		apimeta.RemoveStatusCondition(&status.Conditions, v1alpha2.ConditionTypeInstalled)
	}

	if s.ReadyCondition != nil {
		apimeta.SetStatusCondition(&status.Conditions, *s.ReadyCondition)
	} else {
		apimeta.RemoveStatusCondition(&status.Conditions, v1alpha2.ConditionTypeReady)
	}

	if s.UninstallCondition != nil {
		apimeta.SetStatusCondition(&status.Conditions, *s.UninstallCondition)
	} else {
		apimeta.RemoveStatusCondition(&status.Conditions, v1alpha2.ConditionTypeUninstalled)
	}
}
