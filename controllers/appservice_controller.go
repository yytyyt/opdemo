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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appv1beta1 "github.com/yyt/opdemo/api/v1beta1"
)

var (
	oldSpecAnnotation = "old/spec"
)

// AppServiceReconciler reconciles a AppService object
type AppServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
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
func (r *AppServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

	// 得到AppService 过后去创建对应得Deployment 和Service
	// 创建就得去判断是否存在：存在就忽略 不存在就去创建
	// 更新  合并为 CreateOrUpdate

	// 调谐 获取到当前得一个状态 然后和我们期望得状态进行对比就可以了
	// CreateOrUpdate Deployment
	var deploy appsv1.Deployment
	deploy.Name = appService.Name
	deploy.Namespace = appService.Namespace
	operationResult, err := ctrl.CreateOrUpdate(ctx, r.Client, &deploy, func() error {
		// 调谐必须在这个函数中去实现
		MutateDeployment(&appService, &deploy)
		return controllerutil.SetControllerReference(&appService, &deploy, r.Scheme)

	})
	if err != nil {
		return ctrl.Result{}, nil
	}
	logger.Info("CreateOrUpdate", "Deployment", operationResult)

	// CreateOrUpdate Service
	var svc corev1.Service
	svc.Name = appService.Name
	svc.Namespace = appService.Namespace
	operationResult, err = ctrl.CreateOrUpdate(ctx, r.Client, &svc, func() error {
		// 调谐必须在这个函数中去实现
		MutateService(&appService, &svc)
		return controllerutil.SetControllerReference(&appService, &svc, r.Scheme)

	})
	if err != nil {
		return ctrl.Result{}, nil
	}
	logger.Info("CreateOrUpdate", "Service", operationResult)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1beta1.AppService{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
