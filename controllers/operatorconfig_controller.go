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
	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"os"
	"strings"

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

//+kubebuilder:rbac:groups=crane.konveyor.io,resources=operatorconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crane.konveyor.io,resources=operatorconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crane.konveyor.io,resources=operatorconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=tekton.dev,resources=clustertasks,verbs=get;list;watch;create;update;patch;delete

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

	// your logic here

	err := createClusterTasks(ctx, log)

	if err != nil {
		log.Error(err, "Error creating cluster tasks")
	} else {
		log.Info("cluster task created")
	}
	return ctrl.Result{}, err
}

func createClusterTasks(ctx context.Context, log logr.Logger) error {

	clustertasks := v1beta1.ClusterTask{}
	var err error

	dir, err := os.Getwd()
	if err != nil {
		log.Error(err, "error getting working dir")
		return err
	}
	log.Info(dir)

	var data []byte
	data, err = ioutil.ReadFile("clustertasks.yaml")
	if err != nil {
		return err
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	clientset, err := versioned.NewForConfig(config)
	if err != nil {
		return err
	}

	log.Info(string(data))

	for _, doc := range strings.Split(string(data), "---") {
		err = yaml.Unmarshal([]byte(doc), &clustertasks)
		if err != nil {
			return err
		}
		obj, err := clientset.TektonV1beta1().ClusterTasks().Create(ctx, &clustertasks, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		log.Info("Cluster task created", obj.Name, obj.Namespace)
	}

	//clusterTask := v1beta1.ClusterTask{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name: "crane-apply",
	//		Annotations: map[string]string{
	//			"migration.openshift.io/run-after": "crane-transform",
	//			"description":                      "This is where a really long-form explanation of what is happening in crane-apply ClusterTask would go.",
	//		},
	//	},
	//	Spec: v1beta1.TaskSpec{
	//		Steps: []v1beta1.Step{
	//			{
	//				Container: v1.Container{},
	//				Script:    "/crane apply --export-dir=$(workspaces.export.path) --transform-dir=$(workspaces.transform.path) --output-dir=$(workspaces.apply.path) \n find $(workspaces.apply.path)",
	//			},
	//		},
	//		Workspaces: nil,
	//	},
	//}
	//_, err = clientset.TektonV1beta1().ClusterTasks().Create(ctx, &clusterTask, metav1.CreateOptions{})
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OperatorConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cranev1alpha1.OperatorConfig{}).
		Complete(r)
}
