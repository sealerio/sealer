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

package guest

import (
	"testing"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

func TestDefault_Apply(t *testing.T) {
	type args struct {
		Cluster *v1.Cluster
	}
	tests := []struct {
		name    string
		args    args
		wanterr bool
	}{
		{
			name: "Master exec cmd : echo 'guest_test success",
			args: args{
				Cluster: &v1.Cluster{
					Spec: v1.ClusterSpec{
						Image: "kuberentes:v1.18.6",
						SSH: v1.SSH{
							User:     "root",
							Passwd:   "huaijiahui.com",
							Pk:       "",
							PkPasswd: "",
						},
						Masters: v1.Hosts{
							IPList: []string{"192.168.56.104"},
						},
					},
				},
			},
			wanterr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Default := NewGuestManager()
			if err := Default.Apply(tt.args.Cluster); (err != nil) != tt.wanterr {
				t.Errorf("Apply failed, %s", err)
			}
		})
	}
}
