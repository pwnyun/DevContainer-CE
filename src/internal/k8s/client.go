package k8s

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func GetK8sClient() (*kubernetes.Clientset, error) {
	home := os.Getenv("HOME")
	kubeconfig := filepath.Join(home, ".kube", "config", "kube-config.yaml")
	if _, err := os.Stat(kubeconfig); err != nil {
		kubeconfig = ""
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}
