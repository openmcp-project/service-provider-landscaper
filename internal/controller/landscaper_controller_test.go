package controller_test

import (
	"time"

	testutils "github.com/openmcp-project/controller-utils/pkg/testing"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"github.com/openmcp-project/openmcp-operator/lib/clusteraccess"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/openmcp-project/controller-utils/pkg/clusters"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/openmcp-project/service-provider-landscaper/api/v1alpha1"

	lscontroller "github.com/openmcp-project/service-provider-landscaper/internal/controller"
)

const (
	controllerName = "test-controller"
)

func buildTestEnvironmentReconcile(testdataDir string, objectsWithStatus ...client.Object) *testutils.Environment {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clustersv1alpha1.AddToScheme(scheme))
	utilruntime.Must(lsv1alpha1.AddToScheme(scheme))

	return testutils.NewEnvironmentBuilder().
		WithFakeClient(scheme).
		WithInitObjectPath("testdata", testdataDir).
		WithDynamicObjectsWithStatus(objectsWithStatus...).
		WithReconcilerConstructor(func(c client.Client) reconcile.Reconciler {
			permissions := []clustersv1alpha1.PermissionsRequest{
				{
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{"*"},
							Resources: []string{"*"},
							Verbs:     []string{"*"},
						},
					},
				},
			}

			car := clusteraccess.NewClusterAccessReconciler(c, controllerName)
			car.WithMCPScheme(scheme).
				WithWorkloadScheme(scheme).
				WithMCPPermissions(permissions).
				WithWorkloadPermissions(permissions).
				WithRetryInterval(1 * time.Second)

			platformCluster := clusters.NewTestClusterFromClient("platform", c)
			onboardingCluster := clusters.NewTestClusterFromClient("onboarding", c)

			r := &lscontroller.LandscaperReconciler{
				Scheme:                  scheme,
				ClusterAccessReconciler: car,
				PlatformCluster:         platformCluster,
				OnboardingCluster:       onboardingCluster,
			}

			return r
		}).
		Build()
}

var _ = Describe("Landscaper Controller", func() {
	Context("CreateUpdate", func() {
		It("should set the finalizer and the provider config reference", func() {
			env := buildTestEnvironmentReconcile("test-01")

			req := reconcile.Request{
				NamespacedName: client.ObjectKey{
					Name:      "test",
					Namespace: "default",
				},
			}

			env.ShouldReconcile(req, "reconcile should not return an error and set finalizer")

			ls := &lsv1alpha1.Landscaper{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			}

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(ls), ls)).To(Succeed())
			Expect(ls.ObjectMeta.Finalizers).To(ContainElement(lsv1alpha1.LandscaperFinalizer))

			Expect(ls.Status.ProviderConfigRef.Name).To(Equal("default"))
			Expect(ls.Status.Phase).To(Equal(lsv1alpha1.PhaseProgressing))
			Expect(ls.Status.Conditions).To(HaveLen(2))
			Expect(ls.Status.Conditions[0].Type).To(Equal(lsv1alpha1.ConditionTypeInstalled))
			Expect(ls.Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(ls.Status.Conditions[1].Type).To(Equal(lsv1alpha1.ConditionTypeReady))
			Expect(ls.Status.Conditions[1].Status).To(Equal(metav1.ConditionUnknown))
		})

		It("should use an explicit provider config reference", func() {
			env := buildTestEnvironmentReconcile("test-02")

			req := reconcile.Request{
				NamespacedName: client.ObjectKey{
					Name:      "test",
					Namespace: "default",
				},
			}

			env.ShouldReconcile(req, "reconcile should not return an error and set finalizer")

			ls := &lsv1alpha1.Landscaper{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			}

			Expect(env.Client().Get(env.Ctx, client.ObjectKeyFromObject(ls), ls)).To(Succeed())
			Expect(ls.Status.ProviderConfigRef.Name).To(Equal("test"))
		})
	})
})
