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

	"github.com/alibaba/sealer/common"

	corev1 "k8s.io/api/core/v1"

	"github.com/alibaba/sealer/client"
	"github.com/alibaba/sealer/logger"
)

type PodChecker struct {
}

type PodNamespaceStatus struct {
	NamespaceName     string
	RunningCount      uint32
	NotRunningCount   uint32
	PodCount          uint32
	NotRunningPodList []*corev1.Pod
}

var PodNamespaceStatusList []PodNamespaceStatus

func (n *PodChecker) Check() error {
	// check if all the pod is Running
	c, err := client.NewClientSet()
	if err != nil {
		logger.Info("failed to create k8s client  %v", err)
		return nil
	}
	namespacePodList, err := client.ListAllNamespacesPods(c)
	if err != nil {
		return err
	}
	for _, podNamespace := range namespacePodList {
		var runningCount uint32 = 0
		var notRunningCount uint32 = 0
		var podCount uint32
		var notRunningPodList []*corev1.Pod
		for _, pod := range podNamespace.PodList.Items {
			if err := getPodReadyStatus(&pod); err != nil {
				notRunningCount++
				newPod := pod
				notRunningPodList = append(notRunningPodList, &newPod)
			} else {
				runningCount++
			}
		}
		podCount = runningCount + notRunningCount
		podNamespaceStatus := PodNamespaceStatus{
			NamespaceName:     podNamespace.Namespace.Name,
			RunningCount:      runningCount,
			NotRunningCount:   notRunningCount,
			PodCount:          podCount,
			NotRunningPodList: notRunningPodList,
		}
		PodNamespaceStatusList = append(PodNamespaceStatusList, podNamespaceStatus)
	}
	err = n.Output(PodNamespaceStatusList)
	if err != nil {
		return err
	}
	return nil
}

func (n *PodChecker) Output(podNamespaceStatusList []PodNamespaceStatus) error {
	t := template.New("pod_checker")
	t, err := t.Parse(
		`Cluster Pod Status
  {{ range . -}}
  Namespace: {{ .NamespaceName }}
  RunningPod: {{ .RunningCount }}/{{ .PodCount }}
  {{ if (gt .NotRunningCount 0) -}}
  Not Running Pod List:
    {{- range .NotRunningPodList }}
    PodName: {{ .Name }}
    {{- end }}
  {{ end }}
  {{- end }}
`)
	if err != nil {
		panic(err)
	}
	t = template.Must(t, err)
	err = t.Execute(common.StdOut, podNamespaceStatusList)
	if err != nil {
		logger.Error("pod checker template can not excute %s", err)
		return err
	}
	return nil
}

func getPodReadyStatus(pod *corev1.Pod) error {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == "Ready" {
			if condition.Status == "True" {
				return nil
			}
		}
	}
	return &NotFindReadyTypeError{}
}

func NewPodChecker() Checker {
	return &PodChecker{}
}
