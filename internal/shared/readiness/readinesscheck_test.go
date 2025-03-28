package readiness

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Readiness Check Test Suite")
}

var _ = Describe("Readiness Check", func() {

	It("should return true when the readiness check is ready", func() {
		result := NewReadyResult()
		Expect(result.IsReady()).To(BeTrue())
	})

	It("should return false when the readiness check is not ready", func() {
		result := NewNotReadyResult("test message")
		Expect(result.IsReady()).To(BeFalse())
	})

	It("should return false when the readiness check is failed", func() {
		result := NewFailedResult(nil)
		Expect(result.IsReady()).To(BeFalse())
	})

	It("should return the message", func() {
		result := NewNotReadyResult("test message")
		Expect(result.Message()).To(Equal("test message"))
	})

	It("should return the message with multiple messages", func() {
		result := Aggregate(
			NewNotReadyResult("test message 1"),
			NewNotReadyResult("test message 2"),
		)
		Expect(result.Message()).To(Equal("test message 1, test message 2"))
	})

	It("should return true when a deployment is ready", func() {
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Generation: 1,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptr.To[int32](1),
			},
			Status: appsv1.DeploymentStatus{
				ObservedGeneration: 1,
				Replicas:           1,
				UpdatedReplicas:    1,
				AvailableReplicas:  1,
			},
		}
		result := CheckDeployment(deployment)
		Expect(result.IsReady()).To(BeTrue())
	})

	It("should return false when a deployment is not ready", func() {
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Generation: 1,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptr.To[int32](1),
			},
			Status: appsv1.DeploymentStatus{
				ObservedGeneration: 1,
				Replicas:           1,
				UpdatedReplicas:    0,
				AvailableReplicas:  0,
			},
		}
		result := CheckDeployment(deployment)
		Expect(result.IsReady()).To(BeFalse())
	})
})
