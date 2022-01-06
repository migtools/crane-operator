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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cranev1alpha1 "github.com/konveyor/mtk-operator/api/v1alpha1"
)

// OperatorConfigReconciler reconciles a OperatorConfig object
type OperatorConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// rbac.authorization.k8s.io permissions are needed to create namespace limited role and rolebinding to create deployment and service within mtk-operator
//+kubebuilder:rbac:groups=crane.konveyor.io,resources=operatorconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crane.konveyor.io,resources=operatorconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crane.konveyor.io,resources=operatorconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=tekton.dev,resources=clustertasks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",namespace=mtk-operator,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=console.openshift.io,resourceNames=crane-ui-plugin,resources=consoleplugins,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",namespace=mtk-operator,resources=services,verbs=get;list;watch;create;update;patch;delete

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

	err := createClusterTasks(ctx, log)

	if err != nil {
		log.Error(err, "Error creating cluster tasks")
	} else {
		log.Info("All the needed cluster task created")
	}

	err = configureCranePlugin(ctx, log)

	if err != nil {
		log.Error(err, "error configuring crane plugin")
	} else {
		log.Info("Crane ui plugin configured")
	}
	return ctrl.Result{}, err
}

func configureCranePlugin(ctx context.Context, log logr.Logger) error {
	return createResourcesFromFile("crane-ui-plugin.yaml", ctx, log)
}

func createClusterTasks(ctx context.Context, log logr.Logger) error {
	return createResourcesFromFile("manifests.yaml", ctx, log)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OperatorConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cranev1alpha1.OperatorConfig{}).
		Complete(r)
}
