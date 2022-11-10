/*
Copyright 2022 yyt.

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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/util/retry"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appv1beta1 "github.com/yyt/opdemo/api/v1beta1"
)

var (
	oldSpecAnnotationBeat1 = "old/spec"
)

// AppServiceReconciler reconciles a AppService object
type AppServiceReconcilerBeta1 struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.ydzs.io,resources=appservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.ydzs.io,resources=appservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.ydzs.io,resources=appservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AppService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *AppServiceReconcilerBeta1) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 首先我们获取AppService 实例
	var appService appv1beta1.AppService
	err := r.Client.Get(ctx, req.NamespacedName, &appService)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			// 返回err 会重新入队列获取
			return ctrl.Result{}, err
		}
		// 在删除一个不存在得对象得时候 可能会报not-found得错误
		// 这种情况不需要重新入队列修复
		return ctrl.Result{}, nil
	}

	// 当前得对象标记为了删除
	if appService.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}
	// 如果不存在关联得资源  应该去创建
	// 如果存在关联得资源 判断是否需要更新
	deploy := &appsv1.Deployment{}
	if err = r.Client.Get(ctx, req.NamespacedName, deploy); err != nil && errors.IsNotFound(err) {
		// 关联 Annotations
		bytes, err := json.Marshal(appService.Spec)
		if err != nil {
			return ctrl.Result{}, err
		}
		if appService.Annotations != nil {
			appService.Annotations[oldSpecAnnotation] = string(bytes)
		} else {
			appService.Annotations = map[string]string{
				oldSpecAnnotation: string(bytes),
			}
		}
		logger.Info(string(bytes))
		// 重新更新AppService
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Client.Update(ctx, &appService)
		}); err != nil {
			return ctrl.Result{}, err
		}

		// Deployment  不存在 创建关联得资源
		newDeploy := NewDeploy(&appService)
		if err = r.Client.Create(ctx, newDeploy); err != nil {
			return ctrl.Result{}, err
		}
		// 直接创建 Service
		newService := NewService(&appService)
		if err = r.Client.Create(ctx, newService); err != nil {
			return ctrl.Result{}, err
		}

		// 创建成功
		return ctrl.Result{}, nil
	}

	// TODO 更新 需要判断是否需要更新 （YAML 文件是否发生了变化）
	// yaml  -> old yaml  我们可以从annotations 里面去获取
	oldSpec := appv1beta1.AppServiceSpec{}
	if err = json.Unmarshal([]byte(appService.Annotations[oldSpecAnnotation]), &oldSpec); err != nil {
		return ctrl.Result{}, err
	}

	// 新旧得对象进行比较  如果不一致就应该去更新
	if !reflect.DeepEqual(appService.Spec, oldSpec) {
		// 应该去更新关联资源
		newDeploy := NewDeploy(&appService)
		oldDeploy := &appsv1.Deployment{}
		if err = r.Client.Get(ctx, req.NamespacedName, oldDeploy); err != nil {
			return ctrl.Result{}, err
		}
		oldDeploy.Spec = newDeploy.Spec
		// 直接去更新oldDeploy
		// 注意： 一般情况下 不会直接调用 update 进行更新 应该可能这个deployment被别的控制器 watch  防止多个控制器更新 造成版本不一致
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Client.Update(ctx, oldDeploy)

		}); err != nil {
			return ctrl.Result{}, err
		}

		// 更新service
		newService := NewService(&appService)
		oldService := &corev1.Service{}

		if err = r.Client.Get(ctx, req.NamespacedName, oldService); err != nil {
			return ctrl.Result{}, err
		}
		// 需要指定ClusterIp为之前得  否则更新会报错
		newService.Spec.ClusterIP = oldService.Spec.ClusterIP
		oldService.Spec = newService.Spec
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Client.Update(ctx, oldService)

		}); err != nil {
			return ctrl.Result{}, err
		}

	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppServiceReconcilerBeta1) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1beta1.AppService{}).
		Complete(r)
}
