// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8s

import (
	"context"
	"path/filepath"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/sealerio/sealer/common"
)

type Client struct {
	client *kubernetes.Clientset
}

type NamespacePod struct {
	Namespace v1.Namespace
	PodList   *v1.PodList
}

type NamespaceSvc struct {
	Namespace   v1.Namespace
	ServiceList *v1.ServiceList
}

func NewK8sClient() (*Client, error) {
	kubeconfig := filepath.Join(common.DefaultKubeConfigDir(), "config")
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build kube config")
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		client: clientSet,
	}, nil
}

func (c *Client) ConfigMap(ns string) v12.ConfigMapInterface {
	return c.client.CoreV1().ConfigMaps(ns)
}

func (c *Client) ListNodes() (*v1.NodeList, error) {
	nodes, err := c.client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cluster nodes")
	}
	return nodes, nil
}

func (c *Client) DeleteNode(name string) error {
	if err := c.client.CoreV1().Nodes().Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "failed to delete cluster node(%s)", name)
	}
	return nil
}

func (c *Client) listNamespaces() (*v1.NamespaceList, error) {
	namespaceList, err := c.client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get namespaces")
	}
	return namespaceList, nil
}

func (c *Client) ListAllNamespacesPods() ([]*NamespacePod, error) {
	namespaceList, err := c.listNamespaces()
	if err != nil {
		return nil, err
	}
	var namespacePodList []*NamespacePod
	for _, ns := range namespaceList.Items {
		pods, err := c.client.CoreV1().Pods(ns.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get all namespace pods")
		}
		namespacePod := NamespacePod{
			Namespace: ns,
			PodList:   pods,
		}
		namespacePodList = append(namespacePodList, &namespacePod)
	}

	return namespacePodList, nil
}

func (c *Client) ListAllNamespacesSvcs() ([]*NamespaceSvc, error) {
	namespaceList, err := c.listNamespaces()
	if err != nil {
		return nil, err
	}
	var namespaceSvcList []*NamespaceSvc
	for _, ns := range namespaceList.Items {
		svcs, err := c.client.CoreV1().Services(ns.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get all namespace pods")
		}
		namespaceSvc := NamespaceSvc{
			Namespace:   ns,
			ServiceList: svcs,
		}
		namespaceSvcList = append(namespaceSvcList, &namespaceSvc)
	}
	return namespaceSvcList, nil
}

func (c *Client) GetEndpointsList(namespace string) (*v1.EndpointsList, error) {
	endpointsList, err := c.client.CoreV1().Endpoints(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get endpoint in namespace %s", namespace)
	}
	return endpointsList, nil
}
