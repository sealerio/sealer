package testhelper

import (
	"context"
	"path/filepath"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const DefaultKubeconfigFile = "/root/.kube/config"

func NewClientSet() (*kubernetes.Clientset, error) {
	kubeconfig := DefaultKubeconfigFile
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "new kube build config failed")
	}

	return kubernetes.NewForConfig(config)
}

func ListNodes(client *kubernetes.Clientset) (*v1.NodeList, error) {
	nodes, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster nodes failed")
	}
	return nodes, nil
}

func ListPods(client *kubernetes.Clientset) (*v1.PodList, error) {
	pods, err := client.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster pods failed")
	}
	return pods, nil
}
