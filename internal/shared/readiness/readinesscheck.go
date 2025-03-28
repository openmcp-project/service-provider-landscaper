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

func Ready() CheckResult {
	return CheckResult{}
}

func NotReady(message string) CheckResult {
	return CheckResult{message}
}

func CheckFailed(err error) CheckResult {
	return NotReady(fmt.Sprintf("readiness check failed: %v", err))
}

func Aggregate(results ...CheckResult) CheckResult {
	return slices.Concat(results...)
}

// CheckDeployment checks the readiness of a deployment.
func CheckDeployment(dp *appsv1.Deployment) CheckResult {
	if dp.Status.ObservedGeneration < dp.Generation {
		return NotReady(fmt.Sprintf("deployment %s/%s not ready: observed generation outdated", dp.Namespace, dp.Name))
	}

	var specReplicas int32 = 0
	if dp.Spec.Replicas != nil {
		specReplicas = *dp.Spec.Replicas
	}

	if specReplicas != dp.Status.Replicas || specReplicas != dp.Status.UpdatedReplicas || specReplicas != dp.Status.AvailableReplicas {
		return NotReady(fmt.Sprintf("deployment %s/%s not ready: not enough ready replicas (%d/%d)", dp.Namespace, dp.Name, dp.Status.AvailableReplicas, specReplicas))
	}

	return Ready()
}
