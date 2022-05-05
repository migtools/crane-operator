/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	cranev1alpha1 "github.com/konveyor/crane-operator/api/v1alpha1"
	consolev1alpha1 "github.com/openshift/api/console/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	Finalizer                = "openshift.konveyor.crane"
	OwnerConfigName          = "openshift-migration"
	InvalidNameConditionType = "InvalidName"
)

// An operand, we are defining as:
// 1. the path to the manifest to deploy the operand
// 2. the operands image
type operand struct {
	path    string
	imageFn ImageFunction
}

// operands is the set of components being managed by this operator
var operands = []operand{
	{
		path:    "crane-reverse-proxy.yaml",
		imageFn: CraneReverseProxyImage,
	},
	{
		path:    "crane-secret-service.yaml",
		imageFn: CraneSecretServiceImage,
	},
	{
		path:    "crane-ui-plugin.yaml",
		imageFn: CraneUIPluginImage,
	},
	{
		path:    "crane-runner.yaml",
		imageFn: CraneRunnerImage,
	},
}

// OperatorConfigReconciler reconciles a OperatorConfig object
type OperatorConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// rbac.authorization.k8s.io permissions are needed to create namespace limited role and rolebinding to create deployment and service within openshift-migration
//+kubebuilder:rbac:groups=crane.konveyor.io,resources=operatorconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crane.konveyor.io,resources=operatorconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crane.konveyor.io,resources=operatorconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=tekton.dev,resources=clustertasks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",namespace=openshift-migration,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=console.openshift.io,resources=consoleplugins,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",namespace=openshift-migration,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=route.openshift.io,namespace=openshift-migration,resources=routes,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OperatorConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *OperatorConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	// Fetch the OperatorConfig operatorConfig
	operatorConfig := &cranev1alpha1.OperatorConfig{}
	if err := r.Get(ctx, req.NamespacedName, operatorConfig); err != nil {
		// Error reading the object - requeue the request.
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("OperatorConfig resource not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get OperatorConfig")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if operatorConfig.Name != OwnerConfigName {
		reason := fmt.Sprintf("Invalid name (%s)", operatorConfig.Name)
		msg := fmt.Sprintf("Only one OperatorConfig supported per cluster and must be named '%s'", OwnerConfigName)
		log.Info(fmt.Sprintf("%s: %s", reason, msg))
		if meta.FindStatusCondition(operatorConfig.Status.Conditions, InvalidNameConditionType) == nil {
			meta.SetStatusCondition(&operatorConfig.Status.Conditions, metav1.Condition{
				Type:               InvalidNameConditionType,
				Status:             metav1.ConditionTrue,
				Reason:             "NonStandardNameConfigured",
				Message:            fmt.Sprintf("%s: %s", reason, msg),
				LastTransitionTime: metav1.Time{Time: time.Now()},
			})
			err := r.Status().Update(ctx, operatorConfig)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if meta.FindStatusCondition(operatorConfig.Status.Conditions, InvalidNameConditionType) == nil {
		meta.SetStatusCondition(&operatorConfig.Status.Conditions, metav1.Condition{
			Type:               InvalidNameConditionType,
			Status:             metav1.ConditionFalse,
			Reason:             "StandardNameFound",
			Message:            fmt.Sprintf("Valid name (%s)", operatorConfig.Name),
			LastTransitionTime: metav1.Time{Time: time.Now()},
		})
		err := r.Status().Update(ctx, operatorConfig)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Add Finalizer for this CR
	if !controllerutil.ContainsFinalizer(operatorConfig, Finalizer) {
		controllerutil.AddFinalizer(operatorConfig, Finalizer)
		err := r.Update(ctx, operatorConfig)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if operatorConfig.DeletionTimestamp != nil {
		// clean up
		err := r.cleanUpResources(ctx)
		if err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(operatorConfig, Finalizer)
		err = r.Update(ctx, operatorConfig)
		if err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Clean up successful")
		return ctrl.Result{}, nil
	}

	for _, o := range operands {
		err := r.reconcileOperand(o, ctx, log, operatorConfig)
		if err != nil {
			log.Error(err, "Error creating resources")
			return ctrl.Result{Requeue: true}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *OperatorConfigReconciler) cleanUpResources(ctx context.Context) error {
	for _, o := range operands {
		err := r.deleteOperand(o, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *OperatorConfigReconciler) reconcileOperand(o operand, ctx context.Context, log logr.Logger, operatorConfig *cranev1alpha1.OperatorConfig) error {
	var decoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	data, err := getResources(o.path)
	if err != nil {
		return err
	}

	reconcilersForGVK := map[string]func(resource *unstructured.Unstructured, ctx context.Context, imageFn ImageFunction, log logr.Logger, oc *cranev1alpha1.OperatorConfig) error{
		"Deployment":    r.reconcileDeployment,
		"Service":       r.reconcileService,
		"ConfigMap":     r.reconcileConfigMap,
		"ClusterTask":   r.reconcileClusterTask,
		"ConsolePlugin": r.reconcileConsolePlugin,
	}

	for _, resource := range data {
		if len(resource) > 0 {
			obj := unstructured.Unstructured{}
			_, gvk, err := decoder.Decode([]byte(resource), nil, &obj)
			if err != nil {
				return err
			}

			if reconcile, ok := reconcilersForGVK[gvk.Kind]; ok {
				err := reconcile(&obj, ctx, o.imageFn, log, operatorConfig)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf(fmt.Sprintf("Kind %s is not managed by the operator, check input yamls and make sure all the input are in desired state", gvk.Kind))
			}
		}
	}

	return nil
}

func (r *OperatorConfigReconciler) reconcileDeployment(resource *unstructured.Unstructured, ctx context.Context, imageFn ImageFunction, log logr.Logger, oc *cranev1alpha1.OperatorConfig) error {
	var obj appsv1.Deployment
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(resource.UnstructuredContent(), &obj)
	if err != nil {
		return err
	}

	deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: obj.Name, Namespace: obj.Namespace}}
	op, err := controllerutil.CreateOrPatch(context.TODO(), r.Client, deploy, func() error {
		if deploy.ObjectMeta.CreationTimestamp.IsZero() {
			deploy.Spec.Selector = obj.Spec.Selector
		}

		err = controllerutil.SetControllerReference(oc, deploy, r.Scheme)
		if err != nil {
			return err
		}

		deploy.Spec.Template = obj.Spec.Template
		if len(obj.Labels) > 0 {
			deploy.Labels = obj.Labels
		}
		if len(obj.Annotations) > 0 {
			deploy.Annotations = obj.Annotations
		}

		// Override each of the images in the deployment spec.
		// If, in the future, we have multiple image in a single component we will
		// need a new approach.
		for i := range deploy.Spec.Template.Spec.Containers {
			deploy.Spec.Template.Spec.Containers[i].Image = imageFn()
		}
		return nil
	})
	if err != nil {
		return err
	} else {
		log.Info("Deployment successfully reconciled", "operation", op)
	}

	return nil
}

func (r *OperatorConfigReconciler) reconcileService(resource *unstructured.Unstructured, ctx context.Context, imageFn ImageFunction, log logr.Logger, oc *cranev1alpha1.OperatorConfig) error {
	var obj corev1.Service
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(resource.UnstructuredContent(), &obj)
	if err != nil {
		return err
	}

	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: obj.Namespace, Name: obj.Name}}
	op, err := controllerutil.CreateOrPatch(context.TODO(), r.Client, service, func() error {
		if service.ObjectMeta.CreationTimestamp.IsZero() {
			service.Spec.Selector = obj.Spec.Selector
		}

		err = controllerutil.SetControllerReference(oc, service, r.Scheme)
		if err != nil {
			return err
		}

		service.Spec = obj.Spec
		if len(obj.Labels) > 0 {
			service.Labels = obj.Labels
		}
		if len(obj.Annotations) > 0 {
			service.Annotations = obj.Annotations
		}
		return nil
	})
	if err != nil {
		return err
	} else {
		log.Info("Service successfully reconciled", "operation", op)
	}

	return nil
}

func (r *OperatorConfigReconciler) reconcileConfigMap(resource *unstructured.Unstructured, ctx context.Context, imageFn ImageFunction, log logr.Logger, oc *cranev1alpha1.OperatorConfig) error {
	var obj corev1.ConfigMap
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(resource.UnstructuredContent(), &obj)
	if err != nil {
		return err
	}

	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: obj.Namespace, Name: obj.Name}}
	op, err := controllerutil.CreateOrPatch(context.TODO(), r.Client, configMap, func() error {
		err = controllerutil.SetControllerReference(oc, configMap, r.Scheme)
		if err != nil {
			return err
		}

		configMap.Data = obj.Data
		if len(obj.Labels) > 0 {
			configMap.Labels = obj.Labels
		}
		if len(obj.Annotations) > 0 {
			configMap.Annotations = obj.Annotations
		}
		return nil
	})
	if err != nil {
		return err
	} else {
		log.Info("ConfigMap successfully reconciled", "operation", op)
	}

	return nil
}

func (r *OperatorConfigReconciler) reconcileClusterTask(resource *unstructured.Unstructured, ctx context.Context, imageFn ImageFunction, log logr.Logger, oc *cranev1alpha1.OperatorConfig) error {
	var obj pipelinev1beta1.ClusterTask
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(resource.UnstructuredContent(), &obj)
	if err != nil {
		return err
	}

	clusterTask := &pipelinev1beta1.ClusterTask{ObjectMeta: metav1.ObjectMeta{Namespace: obj.Namespace, Name: obj.Name}}
	op, err := controllerutil.CreateOrPatch(context.TODO(), r.Client, clusterTask, func() error {
		err = controllerutil.SetControllerReference(oc, clusterTask, r.Scheme)
		if err != nil {
			return err
		}

		clusterTask.Spec = obj.Spec
		if len(obj.Labels) > 0 {
			clusterTask.Labels = obj.Labels
		}
		if len(obj.Annotations) > 0 {
			clusterTask.Annotations = obj.Annotations
		}

		for i := range clusterTask.Spec.Steps {
			clusterTask.Spec.Steps[i].Image = imageFn()
		}
		return nil
	})
	if err != nil {
		return err
	} else {
		log.Info("ClusterTask successfully reconciled", "operation", op)
	}

	return nil
}

func (r *OperatorConfigReconciler) reconcileConsolePlugin(resource *unstructured.Unstructured, ctx context.Context, imageFn ImageFunction, log logr.Logger, oc *cranev1alpha1.OperatorConfig) error {
	var obj consolev1alpha1.ConsolePlugin
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(resource.UnstructuredContent(), &obj)
	if err != nil {
		return err
	}

	consolePlugin := &consolev1alpha1.ConsolePlugin{ObjectMeta: metav1.ObjectMeta{Namespace: obj.Namespace, Name: obj.Name}}
	op, err := controllerutil.CreateOrPatch(context.TODO(), r.Client, consolePlugin, func() error {
		err = controllerutil.SetControllerReference(oc, consolePlugin, r.Scheme)
		if err != nil {
			return err
		}

		consolePlugin.Spec = obj.Spec
		if len(obj.Labels) > 0 {
			consolePlugin.Labels = obj.Labels
		}
		if len(obj.Annotations) > 0 {
			consolePlugin.Annotations = obj.Annotations
		}
		return nil
	})
	if err != nil {
		return err
	} else {
		log.Info("ConsolePlugin successfully reconciled", "operation", op)
	}

	return nil
}

func (r *OperatorConfigReconciler) deleteOperand(o operand, ctx context.Context) error {
	var decoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	data, err := getResources(o.path)
	if err != nil {
		return err
	}

	for _, resource := range data {
		if len(resource) > 0 {
			obj := &unstructured.Unstructured{}
			_, _, err := decoder.Decode([]byte(resource), nil, obj)
			if err != nil {
				return err
			}

			if err = r.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj); err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
			}

			if controllerutil.ContainsFinalizer(obj, Finalizer) {
				controllerutil.RemoveFinalizer(obj, Finalizer)
				if err = r.Update(ctx, obj); err != nil {
					return err
				}
			}

			err = r.Delete(ctx, obj)
			if err != nil && !(errors.IsGone(err) || errors.IsNotFound(err)) {
				return err
			}
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OperatorConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cranev1alpha1.OperatorConfig{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&consolev1alpha1.ConsolePlugin{}).
		Owns(&pipelinev1beta1.ClusterTask{}).
		Complete(r)
}
