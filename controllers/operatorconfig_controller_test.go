package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"

	cranev1alpha1 "github.com/konveyor/mtrho-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("Operator Config", func() {
	var deploy *appsv1.Deployment
	var deplSpec appsv1.DeploymentSpec
	var deplKey types.NamespacedName
	var r *OperatorConfigReconciler
	var oc *cranev1alpha1.OperatorConfig
	var or metav1.OwnerReference

	BeforeEach(func() {
		oc = &cranev1alpha1.OperatorConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:   OwnerConfigName,
				Labels: map[string]string{"foo": "bar"},
				UID:    "test",
			},
		}
		r = &OperatorConfigReconciler{Client: c, Scheme: scheme.Scheme}
		or = metav1.OwnerReference{
			UID: oc.UID,
		}
		deploy = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:            fmt.Sprintf("deploy-%d", rand.Int31()), //nolint:gosec
				Namespace:       "default",
				OwnerReferences: []metav1.OwnerReference{or},
			},
		}

		deplSpec = appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"foo": "bar"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "busybox",
							Image: "busybox",
						},
					},
				},
			},
		}

		deplKey = types.NamespacedName{
			Name:      deploy.Name,
			Namespace: deploy.Namespace,
		}

	})

	It("creates a new object if one doesn't exists", func() {
		deploy.Spec = deplSpec
		tmp, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deploy)
		Expect(err).NotTo(HaveOccurred())

		un := unstructured.Unstructured{Object: tmp}
		Expect(err).NotTo(HaveOccurred())
		err = r.reconcileDeployment(un.DeepCopy(), context.TODO(), log.FromContext(context.TODO()), oc)

		By("returning no error")
		Expect(err).NotTo(HaveOccurred())

		By("actually having the deployment created")
		fetched := &appsv1.Deployment{}
		Expect(c.Get(context.TODO(), deplKey, fetched)).To(Succeed())

		By("Verifying deployment")
		Expect(fetched.Spec.Template.Spec.Containers).To(HaveLen(1))
		Expect(fetched.Spec.Template.Spec.Containers[0].Name).To(Equal(deplSpec.Template.Spec.Containers[0].Name))
		Expect(fetched.Spec.Template.Spec.Containers[0].Image).To(Equal(deplSpec.Template.Spec.Containers[0].Image))

	})

	It("patches existing object without changes", func() {
		deploy.Spec = deplSpec
		tmp, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deploy)
		Expect(err).NotTo(HaveOccurred())

		un := unstructured.Unstructured{Object: tmp}
		Expect(err).NotTo(HaveOccurred())
		err = r.reconcileDeployment(un.DeepCopy(), context.TODO(), log.FromContext(context.TODO()), oc)
		By("returning no error")
		Expect(err).NotTo(HaveOccurred())

		By("Updating field to make it drift away from desired")
		var scale int32
		scale = 2
		deploy.Spec.Replicas = &scale
		deploy.OwnerReferences = nil
		err = r.Update(context.TODO(), deploy)
		Expect(err).NotTo(HaveOccurred())

		prePatched := &appsv1.Deployment{}
		Expect(c.Get(context.TODO(), deplKey, prePatched)).To(Succeed())

		deploy.Spec = deplSpec
		deploy.OwnerReferences = []metav1.OwnerReference{or}
		tmp, err = runtime.DefaultUnstructuredConverter.ToUnstructured(deploy)
		Expect(err).NotTo(HaveOccurred())

		un = unstructured.Unstructured{Object: tmp}
		Expect(err).NotTo(HaveOccurred())
		err = r.reconcileDeployment(un.DeepCopy(), context.TODO(), log.FromContext(context.TODO()), oc)

		By("returning no error")
		Expect(err).NotTo(HaveOccurred())

		By("actually having the deployment created")
		fetched := &appsv1.Deployment{}
		Expect(c.Get(context.TODO(), deplKey, fetched)).To(Succeed())

		By("Verifying apiserver added fields")
		Expect(prePatched.Spec.Replicas).To(Equal(fetched.Spec.Replicas))

	})

	It("patches existing object with changes", func() {
		deploy.Spec = deplSpec
		tmp, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deploy)
		Expect(err).NotTo(HaveOccurred())

		un := unstructured.Unstructured{Object: tmp}
		Expect(err).NotTo(HaveOccurred())
		err = r.reconcileDeployment(un.DeepCopy(), context.TODO(), log.FromContext(context.TODO()), oc)
		By("returning no error")
		Expect(err).NotTo(HaveOccurred())

		prePatched := &appsv1.Deployment{}
		Expect(c.Get(context.TODO(), deplKey, prePatched)).To(Succeed())

		deploy.Spec.Template.Spec.Containers[0].Name = "busybox-test"
		tmp, err = runtime.DefaultUnstructuredConverter.ToUnstructured(deploy)
		Expect(err).NotTo(HaveOccurred())

		un = unstructured.Unstructured{Object: tmp}
		Expect(err).NotTo(HaveOccurred())

		err = r.reconcileDeployment(un.DeepCopy(), context.TODO(), log.FromContext(context.TODO()), oc)
		By("returning no error")
		Expect(err).NotTo(HaveOccurred())

		By("actually verifying patch occurred")
		fetched := &appsv1.Deployment{}
		Expect(c.Get(context.TODO(), deplKey, fetched)).To(Succeed())
		Expect(reflect.DeepEqual(prePatched, fetched)).To(BeFalse())
	})
})
