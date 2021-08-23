package clusterinfo

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

// GetPodsIP returns the pods IP without duplicates.
func GetPodsIP(ctx context.Context, client corev1client.CoreV1Interface, namespace string) ([]string, error) {
	ipList := []string{}
	podList, err := client.Pods(namespace).List(ctx, metav1.ListOptions{})
	if  err != nil {
		return nil, errors.Wrapf(err, "failed to get pods ip")
	}

	for i, _ := range podList.Items {
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

	for i, _ := range nodeList.Items {
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

func removeDuplicatesAndEmpty(ss []string) []string {
	sort.Strings(ss)
	ret := []string{}
	l := len(ss)
	for i:=0; i < l; i++ {
		if (i > 0 && ss[i] == ss[i-1]) || len(ss[i]) == 0 {
			continue
		}
		ret = append(ret, ss[i])
	}

	return ret
}
