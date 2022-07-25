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
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"net"
	"testing"
)

func TestDumperPlugin_Run(t *testing.T) {
	type fields struct {
		Plugins []v1.Plugin
		Cluster *v2.Cluster
	}
	type args struct {
		hosts []net.IP
		phase Phase
	}

	//TODO cluster is where?
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test",
			fields: fields{
				Plugins: []v1.Plugin{},
				Cluster: &v2.Cluster{},
			},
			args: args{
				hosts: []net.IP{
					net.ParseIP("116.31.96.134"),
					net.ParseIP("116.31.96.135"),
					net.ParseIP("116.31.96.136"),
				},
				phase: "PostInstall | preInit",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PluginsProcessor{
				Plugins: tt.fields.Plugins,
				Cluster: tt.fields.Cluster,
			}
			if err := c.Run(tt.args.hosts, tt.args.phase); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
