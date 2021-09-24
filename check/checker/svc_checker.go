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

package checker

import (
	"text/template"

	"github.com/alibaba/sealer/client/k8s"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	corev1 "k8s.io/api/core/v1"
)

type SvcChecker struct {
	client *k8s.Client
}

type SvcNamespaceStatus struct {
	NamespaceName       string
	ServiceCount        int
	EndpointCount       int
	UnhealthServiceList []string
}

type SvcClusterStatus struct {
	SvcNamespaceStatusList []*SvcNamespaceStatus
}

func (n *SvcChecker) Check() error {
	namespaceSvcList, err := n.client.ListAllNamespacesSvcs()
	var svcNamespaceStatusList []*SvcNamespaceStatus
	if err != nil {
		return err
	}
	for _, svcNamespace := range namespaceSvcList {
		serviceCount := len(svcNamespace.ServiceList.Items)
		var unhaelthService []string
		var endpointCount = 0
		endpointsList, err := n.client.GetEndpointsList(svcNamespace.Namespace.Name)
		if err != nil {
			break
		}
		for _, service := range svcNamespace.ServiceList.Items {
			if IsExistEndpoint(endpointsList, service.Name) {
				endpointCount++
			} else {
				unhaelthService = append(unhaelthService, service.Name)
			}
		}
		svcNamespaceStatus := SvcNamespaceStatus{
			NamespaceName:       svcNamespace.Namespace.Name,
			ServiceCount:        serviceCount,
			EndpointCount:       endpointCount,
			UnhealthServiceList: unhaelthService,
		}
		svcNamespaceStatusList = append(svcNamespaceStatusList, &svcNamespaceStatus)
	}
	err = n.Output(svcNamespaceStatusList)
	if err != nil {
		return err
	}
	return nil
}

func (n *SvcChecker) Output(svcNamespaceStatusList []*SvcNamespaceStatus) error {
	t := template.New("svc_checker")
	t, err := t.Parse(
		`Cluster Service Status
  {{- range . }}
  Namespace: {{ .NamespaceName }}
  HealthService: {{ .EndpointCount }}/{{ .ServiceCount }}
  UnhealthServiceList:
    {{- range .UnhealthServiceList }}
    ServiceName: {{ . }}
    {{- end }}
  {{- end }}
`)
	if err != nil {
		panic(err)
	}
	t = template.Must(t, err)
	err = t.Execute(common.StdOut, svcNamespaceStatusList)
	if err != nil {
		logger.Error("service checker template can not excute %s", err)
		return err
	}
	return nil
}

func IsExistEndpoint(endpointList *corev1.EndpointsList, serviceName string) bool {
	for _, ep := range endpointList.Items {
		if ep.Name == serviceName {
			if len(ep.Subsets) > 0 {
				return true
			}
		}
	}
	return false
}

func NewSvcChecker() (Checker, error) {
	// check if all the node is ready
	c, err := k8s.Newk8sClient()
	if err != nil {
		return nil, err
	}
	return &SvcChecker{
		client: c,
	}, nil
}
