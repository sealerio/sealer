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

package apply

import (
	"github.com/alibaba/sealer/pkg/logger"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

func TestGetCurrentCluster(t *testing.T) {
	tests := []struct {
		name    string
		want    *v1.Cluster
		wantErr bool
	}{
		{
			"test get cluster nodes",
			&v1.Cluster{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec: v1.ClusterSpec{
					Masters: v1.Hosts{
						IPList: []string{},
					},
					Nodes: v1.Hosts{
						IPList: []string{},
					},
				},
				Status: v1.ClusterStatus{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCurrentCluster()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			logger.Info("masters : %v nodes : %v", got.Spec.Masters.IPList, got.Spec.Nodes.IPList)
		})
	}
}
