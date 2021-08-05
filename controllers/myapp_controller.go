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
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/utils/pointer"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	samplev1 "github.com/zoetrope/reconcile-tips/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// MyAppReconciler reconciles a MyApp object
type MyAppReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sample.zoetrope.github.io,resources=myapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sample.zoetrope.github.io,resources=myapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sample.zoetrope.github.io,resources=myapps/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *MyAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("myapp", req.NamespacedName)

	// your logic here
	var myapp samplev1.MyApp
	err := r.Get(ctx, req.NamespacedName, &myapp)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !myapp.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	//err = r.reconcileDeploymentBySSA(ctx, &myapp)
	err = r.reconcileDeploymentBySSAWithClientComparison(ctx, &myapp)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Second}, nil
}

func (r *MyAppReconciler) reconcileDeploymentBySSA(ctx context.Context, myapp *samplev1.MyApp) error {
	dep := appsv1apply.Deployment(myapp.Name+"-nginx", myapp.Namespace).
		WithLabels(map[string]string{"component": "nginx"}).
		WithSpec(appsv1apply.DeploymentSpec().
			WithReplicas(1).
			WithSelector(metav1apply.LabelSelector().WithMatchLabels(map[string]string{"component": "nginx"})))

	var podTemplate *corev1apply.PodTemplateSpecApplyConfiguration
	if myapp.Spec.PodTemplate != nil {
		podTemplate = myapp.Spec.PodTemplate.Template
	} else {
		podTemplate = corev1apply.PodTemplateSpec()
	}
	podTemplate.WithLabels(map[string]string{"component": "nginx"})

	if podTemplate.Spec == nil {
		podTemplate.WithSpec(corev1apply.PodSpec())
	}
	hasNginxContainer := false
	for _, c := range podTemplate.Spec.Containers {
		if *c.Name == "nginx" {
			hasNginxContainer = true
		}
	}
	if !hasNginxContainer {
		podTemplate.Spec.WithContainers(
			corev1apply.Container().WithName("nginx").WithImage("nginx:latest"))
	}
	dep.Spec.WithTemplate(podTemplate)

	err := setControllerReference(myapp, dep, r.Scheme)
	if err != nil {
		return err
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dep)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var orig, updated appsv1.Deployment
	err = r.Get(ctx, client.ObjectKey{Namespace: myapp.Namespace, Name: myapp.Name + "-nginx"}, &orig)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "myapp-operator",
		Force: pointer.Bool(true),
	})
	if err != nil {
		return err
	}
	err = r.Get(ctx, client.ObjectKey{Namespace: myapp.Namespace, Name: myapp.Name + "-nginx"}, &updated)
	if err != nil {
		return err
	}
	diff := cmp.Diff(orig, updated)
	if len(diff) > 0 {
		fmt.Printf("diff: \n%s\n", diff)
	}
	return nil
}

func (r *MyAppReconciler) reconcileDeploymentBySSAWithClientComparison(ctx context.Context, myapp *samplev1.MyApp) error {
	fieldManager := "myapp-operator"

	dep := appsv1apply.Deployment(myapp.Name+"-nginx", myapp.Namespace).
		WithLabels(map[string]string{"component": "nginx"}).
		WithSpec(appsv1apply.DeploymentSpec().
			WithReplicas(1).
			WithSelector(metav1apply.LabelSelector().WithMatchLabels(map[string]string{"component": "nginx"})))

	var podTemplate *corev1apply.PodTemplateSpecApplyConfiguration
	if myapp.Spec.PodTemplate != nil {
		podTemplate = myapp.Spec.PodTemplate.Template
	} else {
		podTemplate = corev1apply.PodTemplateSpec()
	}
	podTemplate.WithLabels(map[string]string{"component": "nginx"})

	if podTemplate.Spec == nil {
		podTemplate.WithSpec(corev1apply.PodSpec())
	}
	hasNginxContainer := false
	for _, c := range podTemplate.Spec.Containers {
		if *c.Name == "nginx" {
			hasNginxContainer = true
		}
	}
	if !hasNginxContainer {
		podTemplate.Spec.WithContainers(
			corev1apply.Container().WithName("nginx").WithImage("nginx:latest"))
	}
	dep.Spec.WithTemplate(podTemplate)

	err := setControllerReference(myapp, dep, r.Scheme)
	if err != nil {
		return err
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dep)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var orig, updated appsv1.Deployment
	err = r.Get(ctx, client.ObjectKey{Namespace: myapp.Namespace, Name: myapp.Name + "-nginx"}, &orig)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	origApplyConfig, err := appsv1apply.ExtractDeployment(&orig, fieldManager)
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(dep, origApplyConfig) {
		fmt.Println("do nothing")
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: fieldManager,
		Force: pointer.Bool(true),
	})
	if err != nil {
		return err
	}
	err = r.Get(ctx, client.ObjectKey{Namespace: myapp.Namespace, Name: myapp.Name + "-nginx"}, &updated)
	if err != nil {
		return err
	}
	diff := cmp.Diff(orig, updated)
	if len(diff) > 0 {
		fmt.Printf("diff: \n%s\n", diff)
	}
	return nil
}

func setControllerReference(myapp *samplev1.MyApp, dep *appsv1apply.DeploymentApplyConfiguration, scheme *runtime.Scheme) error {
	gvk, err := apiutil.GVKForObject(myapp, scheme)
	if err != nil {
		return err
	}
	dep.WithOwnerReferences(metav1apply.OwnerReference().
		WithAPIVersion(gvk.GroupVersion().String()).
		WithKind(gvk.Kind).
		WithName(myapp.Name).
		WithUID(myapp.GetUID()).
		WithBlockOwnerDeletion(true).
		WithController(true))
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&samplev1.MyApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
