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
	"sort"
	"testing"
)

/* func TestGetPodsIP(t *testing.T) {
	restConfig, err := clientcmd.BuildConfigFromFlags("", common.KubeAdminConf)
	if err != nil {
		t.Errorf("failed to get rest config from file %s", common.KubeAdminConf)
	}

	kubeClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Errorf("failed to create kubernetes client from file %s", common.KubeAdminConf)
	}

	tests := []struct {
		testName  string
		namespace string
	}{
		{
			testName:  "default",
			namespace: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			_, err := GetPodsIP(context.Background(), kubeClientSet.CoreV1(), tt.namespace)
			if err != nil {
				t.Errorf("failed to get pods IP")
			}
		})
	}
}

func TestGetNodesIP(t *testing.T) {
	restConfig, err := clientcmd.BuildConfigFromFlags("", common.KubeAdminConf)
	if err != nil {
		t.Errorf("failed to get rest config from file %s", common.KubeAdminConf)
	}

	kubeClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Errorf("failed to create kubernetes client from file %s", common.KubeAdminConf)
	}

	t.Run("GetNodesIP", func(t *testing.T) {
		_, err := GetNodesIP(context.Background(), kubeClientSet.CoreV1())
		if err != nil {
			t.Errorf("failed to get nodes IP")
		}
	})
}


func TestGetDNSServiceAll(t *testing.T) {
	restConfig, err := clientcmd.BuildConfigFromFlags("", common.KubeAdminConf)
	if err != nil {
		t.Errorf("failed to get rest config from file %s", common.KubeAdminConf)
	}

	kubeClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Errorf("failed to create kubernetes client from file %s", common.KubeAdminConf)
	}

	t.Run("GetDNSServiceAll", func(t *testing.T) {
		_, _, _, err := GetDNSServiceAll(context.Background(), kubeClientSet.CoreV1())
		if err != nil {
			t.Errorf("failed to get DNS Service")
		}
	})
}*/

func TestRemoveDuplicatesAndEmpty(t *testing.T) {
	tests := []struct {
		testName string
		ipList   []string
	}{
		{
			testName: "duplicates",
			ipList:   []string{"192.168.1.2", "192.168.1.3", "192.168.1.4", "192.168.1.2"},
		},
		{
			testName: "empty",
			ipList:   []string{"192.168.1.2", "192.168.1.3", "", "192.168.1.5"},
		},
		{
			testName: "duplicatesAndEmpty",
			ipList:   []string{"192.168.1.2", "", "192.168.1.3", "", "192.168.1.4", "192.168.1.2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			ss := removeDuplicatesAndEmpty(tt.ipList)
			sort.Strings(ss)
			for i := 0; i < len(ss)-1; i++ {
				if ss[i] == ss[i+1] || len(ss[i]) == 0 || len(ss[i+1]) == 0 {
					t.Errorf("failed to remove duplicates and empty string")
				}
			}
		})
	}
}
