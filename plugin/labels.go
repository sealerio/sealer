package plugin

import (
	"github.com/alibaba/sealer/client"
	"github.com/alibaba/sealer/logger"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

type LabelsNodes struct {
	clusterName string
	data        map[string][]label
}

type label struct {
	key   string
	value string
}

func NewLabelsNodes(clusterName string) Interface {
	return &LabelsNodes{
		clusterName: clusterName,
		data:        map[string][]label{},
	}
}

func (l LabelsNodes) Run(context Context, phase Phase) {
	if phase != PhasePostInstall {
		logger.Debug("label nodes is PostInstall!")
		return
	}
	l.data = l.formatData(context.Plugin.Spec.Data)

	c, err := client.NewClientSet()
	if err != nil {
		logger.Error("current cluster not found, %v", err)
		return
	}
	nodeList, err := client.ListNodes(c)
	if err != nil {
		logger.Error("current cluster nodes not found, %v", err)
		return
	}
	for _, v := range nodeList.Items {
		internalIP := l.getAddress(v.Status.Addresses)
		labels, ok := l.data[internalIP]
		if ok {
			m := v.GetLabels()
			for _, val := range labels {
				m[val.key] = val.value
			}
			v.SetLabels(m)
			v.SetResourceVersion("")
			_, err := client.UpdateNode(c, &v)
			if err != nil {
				logger.Error("current cluster nodes label failed, %v", err)
			}
		}
	}

}

func (l LabelsNodes) formatData(data string) map[string][]label {
	m := make(map[string][]label)
	items := strings.Split(data, "\n")
	if items == nil || len(items) == 0 {
		logger.Debug("label data is empty!")
		return m
	}
	for _, v := range items {
		ip := strings.Split(v, " ")[0]
		labelStr := strings.Split(strings.Split(v, " ")[1], ",")
		var labels []label
		for _, l := range labelStr {
			labels = append(labels, label{
				key:   strings.Split(l, "=")[0],
				value: strings.Split(l, "=")[1],
			})
		}
		m[ip] = labels
	}
	return m
}

func (l LabelsNodes) getAddress(addresses []corev1.NodeAddress) string {
	for _, v := range addresses {
		if strings.EqualFold(string(v.Type), "InternalIP") {
			return v.Address
		}
	}
	return ""
}
