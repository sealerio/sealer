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

package plugin

import (
	"fmt"
	"github.com/alibaba/sealer/client/k8s"
	"strings"

	"github.com/alibaba/sealer/logger"

	v1 "k8s.io/api/core/v1"
)

/*
labels plugin in Clusterfile:
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: LABEL
spec:
  data: |
     192.168.0.2 ssd=true
     192.168.0.3 ssd=true
     192.168.0.4 ssd=true
     192.168.0.5 ssd=false,hdd=true
     192.168.0.6 ssd=false,hdd=true
     192.168.0.7 ssd=false,hdd=true
---
LabelsNodes.data key = ip
[]lable{{key=ssd,value=false}, {key=hdd,value=true}}
*/
type LabelsNodes struct {
	data   map[string][]label
	client *k8s.Client
}

type label struct {
	key   string
	value string
}

func NewLabelsNodes() (Interface, error) {
	c, err := k8s.Newk8sClient()
	if err != nil {
		return nil, err
	}
	return &LabelsNodes{
		data:   map[string][]label{},
		client: c,
	}, nil
}

func (l LabelsNodes) Run(context Context, phase Phase) error {
	if phase != PhasePostInstall {
		logger.Debug("label nodes is PostInstall!")
		return nil
	}
	l.data = l.formatData(context.Plugin.Spec.Data)

	nodeList, err := l.client.ListNodes()
	if err != nil {
		return fmt.Errorf("current cluster nodes not found, %v", err)
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

			if _, err := l.client.UpdateNode(&v); err != nil {
				return fmt.Errorf("current cluster nodes label failed, %v", err)
			}
		}
	}
	return err
}

func (l LabelsNodes) formatData(data string) map[string][]label {
	m := make(map[string][]label)
	items := strings.Split(data, "\n")
	if len(items) == 0 {
		logger.Debug("label data is empty!")
		return m
	}
	for _, v := range items {
		tmps := strings.Split(v, " ")
		if len(tmps) != 2 {
			//logger.Warn("label data is no-compliance with the rules! label data: %v", v)
			continue
		}
		ip := tmps[0]
		labelStr := strings.Split(tmps[1], ",")
		if len(labelStr) == 0 {
			logger.Warn("label data is no-compliance with the rules! label data: %v", v)
			continue
		}
		var labels []label
		for _, l := range labelStr {
			tmp := strings.Split(l, "=")
			if len(tmp) != 2 {
				logger.Warn("label data is no-compliance with the rules! label data: %v", l)
				continue
			}
			labels = append(labels, label{
				key:   tmp[0],
				value: tmp[1],
			})
		}
		m[ip] = labels
	}
	return m
}

func (l LabelsNodes) getAddress(addresses []v1.NodeAddress) string {
	for _, v := range addresses {
		if strings.EqualFold(string(v.Type), "InternalIP") {
			return v.Address
		}
	}
	return ""
}
