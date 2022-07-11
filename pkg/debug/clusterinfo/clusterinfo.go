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

package clusterinfo

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

// GetPodsIP returns the pods IP without duplicates.
func GetPodsIP(ctx context.Context, client corev1client.CoreV1Interface, namespace string) ([]string, error) {
	ipList := []string{}
	podList, err := client.Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get pods ip")
	}

	for i := range podList.Items {
		ipList = append(ipList, podList.Items[i].Status.PodIP)
	}

	ipList = removeDuplicatesAndEmpty(ipList)

	return ipList, nil
}

// GetNodesIP returns the nodes IP.
func GetNodesIP(ctx context.Context, client corev1client.CoreV1Interface) ([]string, error) {
	ipList := []string{}
	nodeList, err := client.Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get nodes ip")
	}

	for i := range nodeList.Items {
		ipList = append(ipList, nodeList.Items[i].Status.Addresses[0].Address)
	}

	return ipList, nil
}

// GetDNSServiceAll return the DNS Service domain、DNS service cluster IP、DNS service endpoints.
func GetDNSServiceAll(ctx context.Context, client corev1client.CoreV1Interface) (string, string, []string, error) {
	namespace := "kube-system"
	serviceName := "kube-dns"
	domain := "kube-dns.kube-system.svc"

	service, err := client.Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return "", "", nil, errors.Wrapf(err, "failed to get DNS service")
	}
	serviceClusterIP := service.Spec.ClusterIP

	endpoints, err := client.Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return "", "", nil, errors.Wrapf(err, "failed to get DNS service endpoint")
	}
	endpointsIPs := []string{}
	for _, address := range endpoints.Subsets[0].Addresses {
		endpointsIPs = append(endpointsIPs, address.IP)
	}

	return domain, serviceClusterIP, endpointsIPs, nil
}

func removeDuplicatesAndEmpty(ss []string) (res []string) {
	if len(ss) == 0 {
		return ss
	}

	sMap := map[string]bool{}

	for _, v := range ss {
		if _, ok := sMap[v]; !ok && len(v) != 0 {
			sMap[v] = true
			res = append(res, v)
		}
	}

	return res
}
