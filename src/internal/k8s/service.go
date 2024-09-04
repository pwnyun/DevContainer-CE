package k8s

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceInfo struct {
	NodePort string
}

func CreateService(userID, containerIndex string) (*ServiceInfo, error) {
	clientset, err := GetK8sClient()
	if err != nil {
		return nil, err
	}

	containerName := fmt.Sprintf("vscs-%s-%s", userID, containerIndex)

	// 尝试删除已存在的服务
	err = clientset.CoreV1().Services("default").Delete(context.Background(), containerName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		log.Printf("Failed to delete existing service: %v", err)
		return nil, err
	}

	// 创建新的服务
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: containerName,
			Labels: map[string]string{
				"app": "openvscode-server",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "openvscode-server",
			},
			Ports: []corev1.ServicePort{
				{
					Port:     3000,
					Protocol: corev1.ProtocolTCP,
					NodePort: 30000 + (rand.Int31() % 2767),
				},
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	createdService, err := clientset.CoreV1().Services("default").Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create service: %v", err)
		return nil, err
	}

	nodePort := createdService.Spec.Ports[0].NodePort
	return &ServiceInfo{NodePort: fmt.Sprintf("%d", nodePort)}, nil
}

func DeleteService(userID, containerIndex string) error {
	clientset, err := GetK8sClient()
	if err != nil {
		return err
	}

	containerName := fmt.Sprintf("vscs-%s-%s", userID, containerIndex)

	// 尝试删除服务
	err = clientset.CoreV1().Services("default").Delete(context.Background(), containerName, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Service %s not found, nothing to delete", containerName)
			return nil
		}
		log.Printf("Failed to delete service: %v", err)
		return err
	}

	log.Printf("Service %s successfully deleted", containerName)
	return nil
}
