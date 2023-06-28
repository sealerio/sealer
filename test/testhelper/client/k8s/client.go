// Copyright © 2023 Alibaba Group Holding Ltd.
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
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"regexp"

	"github.com/sealerio/sealer/test/testhelper"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	ReadyStatus = "Ready"
	TRUE        = "True"
	FALSE       = "False"
)

type Client struct {
	client *kubernetes.Clientset
}

type NamespacePod struct {
	Namespace v1.Namespace
	PodList   *v1.PodList
}

type EventPod struct {
	Reason    string
	Message   string
	Count     int32
	Type      string
	Action    string
	Namespace string
}

func NewK8sClient(sshClient *testhelper.SSHClient) (*Client, error) {
	kubeconfigPath := filepath.Join("/root", ".kube", "config")

	data := testhelper.GetRemoteFileData(sshClient, kubeconfigPath)
	reg := regexp.MustCompile(`server: https://(.*):6443`)
	data = reg.ReplaceAll(data, []byte(fmt.Sprintf("server: https://%s:6443", sshClient.RemoteHostIP.String())))
	config, err := clientcmd.RESTConfigFromKubeConfig(data)

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

func (c *Client) ListNodes() (*v1.NodeList, error) {
	nodes, err := c.client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cluster nodes")
	}
	return nodes, nil
}

func (c *Client) listNamespaces() (*v1.NamespaceList, error) {
	namespaceList, err := c.client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get namespaces")
	}
	return namespaceList, nil
}

func (c *Client) ListNodesByLabel(label string) (*v1.NodeList, error) {
	nodes, err := c.client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cluster nodes")
	}
	return nodes, nil
}

func (c *Client) ListNodeIPByLabel(label string) ([]net.IP, error) {
	var ips []net.IP
	nodes, err := c.ListNodesByLabel(label)
	if err != nil {
		return nil, err
	}
	for _, node := range nodes.Items {
		for _, v := range node.Status.Addresses {
			if v.Type == v1.NodeInternalIP {
				ips = append(ips, net.ParseIP(v.Address))
			}
		}
	}
	return ips, nil
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

// Check if all pods of kube-system are ready
func (c *Client) CheckAllKubeSystemPodsReady() (bool, error) {
	pods, err := c.client.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get kube-system namespace pods")
	}
	// pods.Items maybe nil
	if len(pods.Items) == 0 {
		return false, nil
	}
	for _, pod := range pods.Items {
		// pod.Status.ContainerStatus == nil because of pod contain initcontainer
		if len(pod.Status.ContainerStatuses) == 0 {
			continue
		}
		if !pod.Status.ContainerStatuses[0].Ready {
			return false, nil
		}
	}
	return true, nil
}

func (c *Client) GetPodLog(namespace, podName string) (string, error) {
	req := c.client.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{})
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return "", err
	}

	defer func() {
		_ = podLogs.Close()
	}()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (c *Client) GetPodEvents(namespace, podName string) ([]v1.Event, error) {
	events, err := c.client.CoreV1().
		Events(namespace).
		List(context.TODO(), metav1.ListOptions{FieldSelector: "involvedObject.name=" + podName, TypeMeta: metav1.TypeMeta{Kind: "Pod"}})
	if err != nil {
		return nil, err
	}
	return events.Items, nil
}

func (c *Client) getPodReadyStatus(pod v1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type != ReadyStatus {
			continue
		}
		if condition.Status == TRUE {
			return true
		}
	}
	return false
}

func (c *Client) GetNotReadyPodEvent() (map[string][]EventPod, error) {
	namespacePodList, err := c.ListAllNamespacesPods()
	if err != nil {
		return nil, err
	}
	result := make(map[string][]EventPod)
	for _, podNamespace := range namespacePodList {
		for _, pod := range podNamespace.PodList.Items {
			if c.getPodReadyStatus(pod) {
				continue
			}
			events, err := c.GetPodEvents(podNamespace.Namespace.Name, pod.Name)
			if err != nil {
				return nil, err
			}
			var eventpods []EventPod
			for _, event := range events {
				eventpods = append(eventpods, EventPod{
					Reason:    event.Reason,
					Message:   event.Message,
					Count:     event.Count,
					Type:      event.Type,
					Action:    event.Action,
					Namespace: event.Namespace,
				})
			}
			result[pod.Name] = append(result[pod.Name], eventpods...)
		}
	}
	return result, nil
}

// If exist pod not ready, show pod events and logs
func (c *Client) OutputNotReadyPodInfo() error {
	podEvents, err := c.GetNotReadyPodEvent()
	if err != nil {
		return err
	}
	for podName, events := range podEvents {
		fmt.Println("=========================================================================================================================================")
		fmt.Println("PodName: " + podName)
		fmt.Println("******************************************************Events*****************************************************************************")
		var namespace string
		for _, event := range events {
			namespace = event.Namespace
			fmt.Printf("Reason: %s\n", event.Reason)
			fmt.Printf("Message: %s\n", event.Message)
			fmt.Printf("Count: %v\n", event.Count)
			fmt.Printf("Type: %s\n", event.Type)
			fmt.Printf("Action: %s\n", event.Action)
			fmt.Println("------------------------------------------------------------------------------------------------------------------------------------")
		}
		log, err := c.GetPodLog(namespace, podName)
		if err != nil {
			return err
		}
		fmt.Println("********************************************************Log*****************************************************************************")
		fmt.Println(log)
		fmt.Println("=========================================================================================================================================")
	}
	return nil
}

// Check if all nodes are ready
func (c *Client) CheckAllNodeReady() (bool, error) {
	nodes, err := c.ListNodes()
	if err != nil {
		return false, err
	}
	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type != ReadyStatus {
				continue
			}
			if condition.Status == FALSE {
				fmt.Println("********************************************************Node*****************************************************************************")
				fmt.Println(condition.Reason)
				fmt.Println(condition.Message)
				fmt.Println("=========================================================================================================================================")
				return false, nil
			}
		}
	}
	return true, nil
}
