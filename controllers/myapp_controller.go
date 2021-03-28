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
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	samplev1 "github.com/zoetrope/reconcile-tips/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

	//err = r.reconcileDeploymentByOverwriting(ctx, &myapp)
	//err = r.reconcileDeploymentByManualMerge(ctx, &myapp)
	//err = r.reconcileDeploymentByManualMerge2(ctx, &myapp)
	//err = r.reconcileDeploymentByDeepDerivative(ctx, &myapp)
	//err = r.reconcileDeploymentByStrategicMergePatch(ctx, &myapp)
	//err = r.reconcileDeploymentBySSA1(ctx, &myapp)
	//err = r.reconcileDeploymentBySSA2(ctx, &myapp)
	err = r.reconcileDeploymentBySSA3(ctx, &myapp)
	//err = r.reconcileServiceByOverwriting(ctx, &myapp)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Second}, nil
}

func (r *MyAppReconciler) reconcileDeploymentByOverwriting(ctx context.Context, myapp *samplev1.MyApp) error {
	dep := &appsv1.Deployment{}
	dep.Namespace = myapp.Namespace
	dep.Name = myapp.Name + "-nginx"

	var orig, updated *appsv1.Deployment
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, dep, func() error {
		orig = dep.DeepCopy()

		dep.Labels = map[string]string{
			"component": "nginx",
		}
		dep.Spec = appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"component": "nginx",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"component": "nginx",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			},
		}
		updated = dep.DeepCopy()
		return ctrl.SetControllerReference(myapp, dep, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile deployment: %w", err)
	}

	if result != controllerutil.OperationResultNone {
		fmt.Println(cmp.Diff(orig, updated))
	}
	return nil
}

func (r *MyAppReconciler) reconcileServiceByOverwriting(ctx context.Context, myapp *samplev1.MyApp) error {
	svc := &corev1.Service{}
	svc.Namespace = myapp.Namespace
	svc.Name = myapp.Name + "-service"

	var orig, updated *corev1.Service
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, svc, func() error {
		orig = svc.DeepCopy()

		svc.Spec = corev1.ServiceSpec{
			Selector: map[string]string{
				"component": "nginx",
			},
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		}
		updated = svc.DeepCopy()
		return ctrl.SetControllerReference(myapp, svc, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile service: %w", err)
	}

	if result != controllerutil.OperationResultNone {
		fmt.Println(cmp.Diff(orig, updated))
	}
	return nil
}

func (r *MyAppReconciler) reconcileDeploymentByManualMerge(ctx context.Context, myapp *samplev1.MyApp) error {
	dep := &appsv1.Deployment{}
	dep.Namespace = myapp.Namespace
	dep.Name = myapp.Name + "-nginx"

	var orig, updated *appsv1.Deployment
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, dep, func() error {
		orig = dep.DeepCopy()

		if dep.Labels == nil {
			dep.Labels = make(map[string]string)
		}
		dep.Labels["component"] = "nginx"
		dep.Spec.Replicas = pointer.Int32Ptr(1)
		dep.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"component": "nginx",
			},
		}
		dep.Spec.Template.Labels = map[string]string{
			"component": "nginx",
		}
		if len(dep.Spec.Template.Spec.Containers) == 0 {
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
				},
			}
		}
		updated = dep.DeepCopy()
		return ctrl.SetControllerReference(myapp, dep, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile deployment: %w", err)
	}

	if result != controllerutil.OperationResultNone {
		fmt.Println(cmp.Diff(orig, updated))
	}
	return nil
}

func (r *MyAppReconciler) reconcileDeploymentByManualMerge2(ctx context.Context, myapp *samplev1.MyApp) error {
	dep := &appsv1.Deployment{}
	dep.Namespace = myapp.Namespace
	dep.Name = myapp.Name + "-nginx"

	var orig, updated *appsv1.Deployment
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, dep, func() error {
		orig = dep.DeepCopy()

		if dep.Labels == nil {
			dep.Labels = make(map[string]string)
		}
		dep.Labels["component"] = "nginx"
		dep.Spec.Replicas = pointer.Int32Ptr(1)
		dep.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"component": "nginx",
			},
		}

		podTemplate := myapp.Spec.PodTemplate.Template.DeepCopy()
		if podTemplate.Labels == nil {
			podTemplate.Labels = make(map[string]string)
		}
		podTemplate.Labels["component"] = "nginx"
		hasNginxContainer := false
		for _, c := range podTemplate.Spec.Containers {
			if c.Name == "nginx" {
				hasNginxContainer = true
			}
		}
		if !hasNginxContainer {
			podTemplate.Spec.Containers = append(podTemplate.Spec.Containers, corev1.Container{
				Name:  "nginx",
				Image: "nginx:latest",
			})
		}
		for i, c := range podTemplate.Spec.Containers {
			for _, cur := range dep.Spec.Template.Spec.Containers {
				if c.Name == cur.Name {
					if len(c.ImagePullPolicy) == 0 && len(cur.ImagePullPolicy) > 0 {
						podTemplate.Spec.Containers[i].ImagePullPolicy = cur.ImagePullPolicy
					}
					if len(c.TerminationMessagePath) == 0 && len(cur.TerminationMessagePath) > 0 {
						podTemplate.Spec.Containers[i].TerminationMessagePath = cur.TerminationMessagePath
					}
					if len(c.TerminationMessagePolicy) == 0 && len(cur.TerminationMessagePolicy) > 0 {
						podTemplate.Spec.Containers[i].TerminationMessagePolicy = cur.TerminationMessagePolicy
					}
					/* 中略 */
				}
			}
		}
		if len(podTemplate.Spec.RestartPolicy) == 0 && len(dep.Spec.Template.Spec.RestartPolicy) > 0 {
			podTemplate.Spec.RestartPolicy = dep.Spec.Template.Spec.RestartPolicy
		}
		if len(podTemplate.Spec.SchedulerName) == 0 && len(dep.Spec.Template.Spec.SchedulerName) > 0 {
			podTemplate.Spec.SchedulerName = dep.Spec.Template.Spec.SchedulerName
		}
		if len(podTemplate.Spec.DNSPolicy) == 0 && len(dep.Spec.Template.Spec.DNSPolicy) > 0 {
			podTemplate.Spec.DNSPolicy = dep.Spec.Template.Spec.DNSPolicy
		}
		if podTemplate.Spec.TerminationGracePeriodSeconds == nil && dep.Spec.Template.Spec.TerminationGracePeriodSeconds != nil {
			podTemplate.Spec.TerminationGracePeriodSeconds = dep.Spec.Template.Spec.TerminationGracePeriodSeconds
		}
		if podTemplate.Spec.SecurityContext == nil && dep.Spec.Template.Spec.SecurityContext != nil {
			podTemplate.Spec.SecurityContext = dep.Spec.Template.Spec.SecurityContext.DeepCopy()
		}
		/* 中略 */

		podTemplate.DeepCopyInto(&dep.Spec.Template)
		updated = dep.DeepCopy()
		return ctrl.SetControllerReference(myapp, dep, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile deployment: %w", err)
	}

	if result != controllerutil.OperationResultNone {
		fmt.Println(cmp.Diff(orig, updated))
	}
	return nil
}

func (r *MyAppReconciler) reconcileDeploymentByDeepDerivative(ctx context.Context, myapp *samplev1.MyApp) error {
	dep := &appsv1.Deployment{}
	dep.Namespace = myapp.Namespace
	dep.Name = myapp.Name + "-nginx"

	var orig, updated *appsv1.Deployment
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, dep, func() error {
		orig = dep.DeepCopy()

		if dep.Labels == nil {
			dep.Labels = make(map[string]string)
		}
		dep.Labels["component"] = "nginx"
		dep.Spec.Replicas = pointer.Int32Ptr(1)
		dep.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"component": "nginx",
			},
		}

		podTemplate := myapp.Spec.PodTemplate.Template.DeepCopy()
		if podTemplate.Labels == nil {
			podTemplate.Labels = make(map[string]string)
		}
		podTemplate.Labels["component"] = "nginx"
		hasNginxContainer := false
		for _, c := range podTemplate.Spec.Containers {
			if c.Name == "nginx" {
				hasNginxContainer = true
			}
		}
		if !hasNginxContainer {
			podTemplate.Spec.Containers = append(podTemplate.Spec.Containers, corev1.Container{
				Name:  "nginx",
				Image: "nginx:latest",
			})
		}
		if !equality.Semantic.DeepDerivative(*podTemplate, dep.Spec.Template) {
			podTemplate.DeepCopyInto(&dep.Spec.Template)
		}

		updated = dep.DeepCopy()
		return ctrl.SetControllerReference(myapp, dep, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile deployment: %w", err)
	}

	if result != controllerutil.OperationResultNone {
		fmt.Println(cmp.Diff(orig, updated))
	}
	return nil
}

func (r *MyAppReconciler) reconcileDeploymentByStrategicMergePatch(ctx context.Context, myapp *samplev1.MyApp) error {
	dep := &appsv1.Deployment{}
	dep.Namespace = myapp.Namespace
	dep.Name = myapp.Name + "-nginx"

	if dep.Labels == nil {
		dep.Labels = make(map[string]string)
	}
	dep.Labels["component"] = "nginx"
	dep.Spec.Replicas = pointer.Int32Ptr(1)
	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"component": "nginx",
		},
	}
	podTemplate := myapp.Spec.PodTemplate.Template.DeepCopy()
	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}
	podTemplate.Labels["component"] = "nginx"
	hasNginxContainer := false
	for _, c := range podTemplate.Spec.Containers {
		if c.Name == "nginx" {
			hasNginxContainer = true
		}
	}
	if !hasNginxContainer {
		podTemplate.Spec.Containers = append(podTemplate.Spec.Containers, corev1.Container{
			Name:  "nginx",
			Image: "nginx:latest",
		})
	}
	podTemplate.DeepCopyInto(&dep.Spec.Template)
	err := ctrl.SetControllerReference(myapp, dep, r.Scheme)
	if err != nil {
		return err
	}

	depEncoded, err := runtime.Encode(unstructured.UnstructuredJSONScheme, dep)
	if err != nil {
		return err
	}
	if dep.Annotations == nil {
		dep.Annotations = make(map[string]string)
	}
	dep.Annotations["last-applied"] = string(depEncoded)

	var current appsv1.Deployment
	err = r.Get(ctx, client.ObjectKey{Namespace: myapp.Namespace, Name: myapp.Name + "-nginx"}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get deployment: %w", err)
	}
	if errors.IsNotFound(err) {
		err = r.Create(ctx, dep)
		return err
	}
	current.SetCreationTimestamp(metav1.Time{})
	currentBytes, err := json.Marshal(current)
	if err != nil {
		return err
	}

	modified, err := json.Marshal(dep)
	if err != nil {
		return err
	}
	lastApplied := []byte(dep.Annotations["last-applied"])
	if len(lastApplied) == 0 {
		lastApplied = modified
	}

	patchMeta, err := strategicpatch.NewPatchMetaFromStruct(current)
	patch, err := strategicpatch.CreateThreeWayMergePatch(lastApplied, modified, currentBytes, patchMeta, true)
	if err != nil {
		return err
	}
	err = r.Patch(ctx, &current, client.RawPatch(types.StrategicMergePatchType, patch))
	if err != nil {
		return err
	}
	if string(patch) != "{}" {
		fmt.Println(string(patch))
	}
	return nil
}

func (r *MyAppReconciler) reconcileDeploymentBySSA1(ctx context.Context, myapp *samplev1.MyApp) error {
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	depYaml := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: %s
  name: %s-nginx
  labels:
    component: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      component: nginx
  template:
    metadata:
      labels:
        component: nginx
    spec:
      containers:
      - image: nginx:latest
        name: nginx
`, myapp.Namespace, myapp.Name)

	patch := &unstructured.Unstructured{}
	_, _, err := decUnstructured.Decode([]byte(depYaml), nil, patch)
	if err != nil {
		return err
	}
	//err = ctrl.SetControllerReference(myapp, patch, r.Scheme)
	//if err != nil {
	//	return err
	//}

	var orig, updated appsv1.Deployment
	err = r.Get(ctx, client.ObjectKey{Namespace: myapp.Namespace, Name: myapp.Name + "-nginx"}, &orig)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "myapp-operator",
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

func (r *MyAppReconciler) reconcileDeploymentBySSA2(ctx context.Context, myapp *samplev1.MyApp) error {
	dep := &appsv1.Deployment{}
	dep.Kind = "Deployment"
	dep.APIVersion = appsv1.SchemeGroupVersion.String()
	dep.Namespace = myapp.Namespace
	dep.Name = myapp.Name + "-nginx"

	if dep.Labels == nil {
		dep.Labels = make(map[string]string)
	}
	dep.Labels["component"] = "nginx"
	dep.Spec.Replicas = pointer.Int32Ptr(1)
	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"component": "nginx",
		},
	}
	podTemplate := myapp.Spec.PodTemplate.Template.DeepCopy()
	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}
	podTemplate.Labels["component"] = "nginx"
	hasNginxContainer := false
	for _, c := range podTemplate.Spec.Containers {
		if c.Name == "nginx" {
			hasNginxContainer = true
		}
	}
	if !hasNginxContainer {
		podTemplate.Spec.Containers = append(podTemplate.Spec.Containers, corev1.Container{
			Name:  "nginx",
			Image: "nginx:latest",
		})
	}
	podTemplate.DeepCopyInto(&dep.Spec.Template)
	err := ctrl.SetControllerReference(myapp, dep, r.Scheme)
	if err != nil {
		return err
	}

	var orig, updated appsv1.Deployment
	err = r.Get(ctx, client.ObjectKey{Namespace: myapp.Namespace, Name: myapp.Name + "-nginx"}, &orig)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	err = r.Patch(ctx, dep, client.Apply, &client.PatchOptions{
		FieldManager: "myapp-operator",
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

func (r *MyAppReconciler) reconcileDeploymentBySSA3(ctx context.Context, myapp *samplev1.MyApp) error {
	dep := &appsv1.Deployment{}
	dep.Namespace = myapp.Namespace
	dep.Name = myapp.Name + "-nginx"

	if dep.Labels == nil {
		dep.Labels = make(map[string]string)
	}
	dep.Labels["component"] = "nginx"
	dep.Spec.Replicas = pointer.Int32Ptr(1)
	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"component": "nginx",
		},
	}
	podTemplate := myapp.Spec.PodTemplate.Template.DeepCopy()
	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}
	podTemplate.Labels["component"] = "nginx"
	hasNginxContainer := false
	for _, c := range podTemplate.Spec.Containers {
		if c.Name == "nginx" {
			hasNginxContainer = true
		}
	}
	if !hasNginxContainer {
		podTemplate.Spec.Containers = append(podTemplate.Spec.Containers, corev1.Container{
			Name:  "nginx",
			Image: "nginx:latest",
		})
	}
	podTemplate.DeepCopyInto(&dep.Spec.Template)
	err := ctrl.SetControllerReference(myapp, dep, r.Scheme)
	if err != nil {
		return err
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dep)
	if err != nil {
		return err
	}
	delete(obj["metadata"].(map[string]interface{}), "creationTimestamp")
	delete(obj["spec"].(map[string]interface{}), "strategy")
	delete(obj["spec"].(map[string]interface{})["template"].(map[string]interface{}), "creationTimestamp")
	for i, co := range podTemplate.Spec.Containers {
		if len(co.Resources.Limits) == 0 && len(co.Resources.Requests) == 0 {
			delete(obj["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["containers"].([]interface{})[i].(map[string]interface{}), "resources")
		}
	}

	patch := &unstructured.Unstructured{
		Object: obj,
	}
	patch.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	})
	var orig, updated appsv1.Deployment
	err = r.Get(ctx, client.ObjectKey{Namespace: myapp.Namespace, Name: myapp.Name + "-nginx"}, &orig)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "myapp-operator",
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

// SetupWithManager sets up the controller with the Manager.
func (r *MyAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&samplev1.MyApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
