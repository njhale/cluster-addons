package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	discoveryv1alpha1 "sigs.k8s.io/addon-operators/discovery/api/v1alpha1"
	"sigs.k8s.io/addon-operators/discovery/controllers/decorators"
	"sigs.k8s.io/addon-operators/discovery/lib/testobj"
)

var _ = Describe("Addon Reconciler", func() {
	var (
		ctx                            context.Context
		addon                          *discoveryv1alpha1.Addon
		name                           types.NamespacedName
		expectedKey                    string
		expectedComponentLabelSelector *metav1.LabelSelector
	)

	BeforeEach(func() {
		ctx = context.Background()
		addon = newAddon(genName("ghost-")).Addon
		name = types.NamespacedName{Name: addon.GetName()}
		expectedKey = decorators.ComponentLabelKeyPrefix + addon.GetName()
		expectedComponentLabelSelector = &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      expectedKey,
					Operator: metav1.LabelSelectorOpExists,
				},
			},
		}

		Expect(k8sClient.Create(ctx, addon)).To(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Get(ctx, name, addon)).To(Succeed())
		Expect(k8sClient.Delete(ctx, addon, deleteOpts)).To(Succeed())
	})

	Describe("component selection", func() {
		BeforeEach(func() {
			Eventually(func() (*discoveryv1alpha1.Components, error) {
				err := k8sClient.Get(ctx, name, addon)
				return addon.Status.Components, err
			}, timeout, interval).ShouldNot(BeNil())

			Eventually(func() (*metav1.LabelSelector, error) {
				err := k8sClient.Get(ctx, name, addon)
				return addon.Status.Components.LabelSelector, err
			}, timeout, interval).Should(Equal(expectedComponentLabelSelector))
		})

		Context("with no components bearing its label", func() {
			Specify("a status containing no component references", func() {
				Consistently(func() ([]discoveryv1alpha1.RichReference, error) {
					err := k8sClient.Get(ctx, name, addon)
					return addon.Status.Components.Refs, err
				}, timeout, interval).Should(BeEmpty())
			})
		})

		Context("with components bearing its label", func() {
			var (
				objs          []runtime.Object
				expectedRefs  []discoveryv1alpha1.RichReference
				expectedMetas []discoveryv1alpha1.Metadata
				namespace     string
			)

			BeforeEach(func() {
				namespace = genName("ns-")
				objs = testobj.WithLabel(expectedKey, "",
					testobj.WithName(namespace, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
							decorators.ComponentLabelKeyPrefix + "my.friendly.id": `{"namespace": "metadata.name"}`,
						}},
					}),
				)

				for _, obj := range objs {
					Expect(k8sClient.Create(ctx, obj)).To(Succeed())
				}

				expectedRefs = toRefs(scheme, objs...)
				expectedMetas = []discoveryv1alpha1.Metadata{
					{
						Type:    "my.friendly.id",
						Ref:     "test",
						Content: json.RawMessage(fmt.Sprintf(`{"namespace":"%s"}`, namespace)),
					},
				}
			})

			AfterEach(func() {
				for _, obj := range objs {
					Expect(k8sClient.Get(ctx, testobj.NamespacedName(obj), obj)).To(Succeed())
					Expect(k8sClient.Delete(ctx, obj, deleteOpts)).To(Succeed())
				}
			})

			Specify("a status containing its component references", func() {
				Eventually(func() ([]discoveryv1alpha1.RichReference, error) {
					err := k8sClient.Get(ctx, name, addon)
					return addon.Status.Components.Refs, err
				}, timeout, interval).Should(ConsistOf(expectedRefs))
			})

			Specify("a status containing its extracted metadata", func() {
				Eventually(func() ([]discoveryv1alpha1.Metadata, error) {
					err := k8sClient.Get(ctx, name, addon)
					return addon.Status.Metadata, err
				}, timeout, interval).Should(ConsistOf(expectedMetas))
			})

			Context("when new components are labelled", func() {
				BeforeEach(func() {
					saName := &types.NamespacedName{Namespace: namespace, Name: genName("sa-")}
					newObjs := testobj.WithLabel(expectedKey, "",
						testobj.WithNamespacedName(saName, &corev1.ServiceAccount{}),
						testobj.WithName(genName("sa-admin-"), &rbacv1.ClusterRoleBinding{
							Subjects: []rbacv1.Subject{
								{
									Kind:      rbacv1.ServiceAccountKind,
									Name:      saName.Name,
									Namespace: saName.Namespace,
								},
							},
							RoleRef: rbacv1.RoleRef{
								APIGroup: rbacv1.GroupName,
								Kind:     "ClusterRole",
								Name:     "cluster-admin",
							},
						}),
					)

					for _, obj := range newObjs {
						Expect(k8sClient.Create(ctx, obj)).To(Succeed())
					}

					objs = append(objs, newObjs...)
					expectedRefs = append(expectedRefs, toRefs(scheme, newObjs...)...)
				})

				It("should add the component references", func() {
					Eventually(func() ([]discoveryv1alpha1.RichReference, error) {
						err := k8sClient.Get(ctx, name, addon)
						return addon.Status.Components.Refs, err
					}, timeout, interval).Should(ConsistOf(expectedRefs))
				})
			})

			Context("when component labels are removed", func() {
				BeforeEach(func() {
					for _, obj := range testobj.StripLabel(expectedKey, objs...) {
						Expect(k8sClient.Update(ctx, obj)).To(Succeed())
					}
				})

				It("should remove the component references", func() {
					Eventually(func() ([]discoveryv1alpha1.RichReference, error) {
						err := k8sClient.Get(ctx, name, addon)
						return addon.Status.Components.Refs, err
					}, timeout, interval).Should(BeEmpty())
				})
			})
		})
	})

})
