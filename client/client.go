package client

import (
	"context"
	"fmt"
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

func GetNodeByName(client *kubernetes.Clientset, nodeName string) (node *v1.Node, err error) {
	return client.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
}

func IsNodeReady(node v1.Node) bool {
	nodeConditions := node.Status.Conditions
	for _, condition := range nodeConditions {
		if condition.Type == v1.NodeReady && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

func CordonUnCordon(k8sClient *kubernetes.Clientset, nodeName string, cordoned bool) error {
	node, err := GetNodeByName(k8sClient, nodeName)
	if err != nil {
		return err
	}
	if node.Spec.Unschedulable == cordoned {
		return nil
	}
	node.Spec.Unschedulable = cordoned
	_, err = k8sClient.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error setting cordoned state for  %s node err: %v", nodeName, err)
	}
	return nil
}
