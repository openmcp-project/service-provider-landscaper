package readiness

import (
	"fmt"
	"slices"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
)

type CheckResult []string

func (r CheckResult) IsReady() bool {
	return len(r) == 0
}

func (r CheckResult) Message() string {
	return strings.Join(r, ", ")
}

func NewReadyResult() CheckResult {
	return CheckResult{}
}

func NewNotReadyResult(message string) CheckResult {
	return CheckResult{message}
}

func NewFailedResult(err error) CheckResult {
	return NewNotReadyResult(fmt.Sprintf("readiness check failed: %v", err))
}

func Aggregate(results ...CheckResult) CheckResult {
	return slices.Concat(results...)
}

// CheckDeployment checks the readiness of a deployment.
func CheckDeployment(dp *appsv1.Deployment) CheckResult {
	if dp.Status.ObservedGeneration < dp.Generation {
		return NewNotReadyResult(fmt.Sprintf("deployment %s/%s not ready: observed generation outdated", dp.Namespace, dp.Name))
	}

	var specReplicas int32 = 0
	if dp.Spec.Replicas != nil {
		specReplicas = *dp.Spec.Replicas
	}

	if dp.Generation != dp.Status.ObservedGeneration ||
		specReplicas != dp.Status.Replicas ||
		specReplicas != dp.Status.UpdatedReplicas ||
		specReplicas != dp.Status.AvailableReplicas {
		return NewNotReadyResult(fmt.Sprintf("deployment %s/%s is not ready", dp.Namespace, dp.Name))
	}

	return NewReadyResult()
}
