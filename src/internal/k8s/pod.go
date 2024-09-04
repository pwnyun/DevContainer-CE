package k8s

import (
	"context"
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func StartContainer(userID string, containerIndex string, imageName string, secretToken string) error {
	clientset, err := GetK8sClient()
	if err != nil {
		return err
	}

	containerName := fmt.Sprintf("vscs-%s-%s", userID, containerIndex)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: containerName,
			Labels: map[string]string{
				"app": "openvscode-server",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    containerName,
					Image:   imageName,
					Command: []string{"sh", "-c", fmt.Sprintf("exec ${OPENVSCODE_SERVER_ROOT}/bin/openvscode-server --connection-token %s --host 0.0.0.0 --enable-remote-auto-shutdown", secretToken)},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 3000,
						},
					},
				},
			},
		},
	}

	_, err = clientset.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to start container: %v", err)
		return err
	}

	return nil
}

func StopContainer(userID string, containerIndex string) error {
	clientset, err := GetK8sClient()
	if err != nil {
		return err
	}

	containerName := fmt.Sprintf("vscs-%s-%s", userID, containerIndex)

	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: new(int64),
		PropagationPolicy:  &deletePolicy,
	}
	*deleteOptions.GracePeriodSeconds = 0

	err = clientset.CoreV1().Pods("default").Delete(context.Background(), containerName, deleteOptions)
	if err != nil {
		log.Printf("Failed to delete Pod %s: %v", containerName, err)
		return err
	}

	log.Printf("Pod %s successfully deleted", containerName)
	return nil
}
