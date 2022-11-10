package controllers

import (
	appv1beta1 "github.com/yyt/opdemo/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func MutateDeployment(app *appv1beta1.AppService, deploy *appsv1.Deployment) {
	labels := map[string]string{
		"myapp": app.Name,
	}
	selector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: app.Spec.Size,
		Selector: selector,
		Template: corev1.PodTemplateSpec{ // Pod Template
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Containers: newContainers(app),
			},
		},
	}
}

func MutateService(app *appv1beta1.AppService, svc *corev1.Service) {
	labels := map[string]string{
		"myapp": app.Name,
	}

	svc.Spec = corev1.ServiceSpec{
		ClusterIP: svc.Spec.ClusterIP,
		Ports:     app.Spec.Ports,
		Selector:  labels,
		Type:      corev1.ServiceTypeNodePort,
	}
}

func NewDeploy(app *appv1beta1.AppService) *appsv1.Deployment {
	labels := map[string]string{
		"myapp": app.Name,
	}
	selector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            app.Name,
			Namespace:       app.Namespace,
			OwnerReferences: makeOwnerReference(app),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: app.Spec.Size,
			Selector: selector,
			Template: corev1.PodTemplateSpec{ // Pod Template
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: newContainers(app),
				},
			},
		},
		Status: appsv1.DeploymentStatus{},
	}
}

func makeOwnerReference(app *appv1beta1.AppService) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(app, schema.GroupVersionKind{
			Group:   appv1beta1.GroupVersion.Group,
			Version: appv1beta1.GroupVersion.Version,
			Kind:    appv1beta1.Kind,
		}),
	}
}

func newContainers(app *appv1beta1.AppService) []corev1.Container {
	var containerPorts []corev1.ContainerPort
	for _, svcPort := range app.Spec.Ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			ContainerPort: svcPort.TargetPort.IntVal,
		})
	}
	return []corev1.Container{
		{
			Name:      app.Name,
			Image:     app.Spec.Image,
			Env:       app.Spec.Envs,
			Resources: app.Spec.Resources,
			Ports:     containerPorts,
		},
	}
}

func NewService(app *appv1beta1.AppService) *corev1.Service {
	labels := map[string]string{
		"myapp": app.Name,
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            app.Name,
			Namespace:       app.Namespace,
			OwnerReferences: makeOwnerReference(app),
		},
		Spec: corev1.ServiceSpec{
			Ports:    app.Spec.Ports,
			Selector: labels,
			Type:     corev1.ServiceTypeNodePort,
		},
	}
}
