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

package env

import (
	"net"
	"reflect"
	"testing"

	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

func Test_convertEnv(t *testing.T) {
	type args struct {
		envList []string
	}
	tests := []struct {
		name    string
		args    args
		wantEnv map[string]interface{}
	}{
		{
			"test convert env",
			args{envList: []string{"IP=127.0.0.1;127.0.0.2;127.0.0.3", "IP=192.168.0.2", "key=value"}},
			map[string]interface{}{"IP": []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "192.168.0.2"}, "key": "value"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotEnv := ConvertEnv(tt.args.envList); !reflect.DeepEqual(gotEnv, tt.wantEnv) {
				t.Errorf("convertEnv() = %v, want %v", gotEnv, tt.wantEnv)
			}
		})
	}
}

func getTestCluster() *v2.Cluster {
	return &v2.Cluster{
		Spec: v2.ClusterSpec{
			Image: "",
			Env:   []string{"IP=127.0.0.1", "key=value"},
			Hosts: []v2.Host{
				{
					IPS:   []net.IP{net.ParseIP("192.168.0.2"), net.ParseIP("192.168.0.3"), net.ParseIP("192.168.0.4")},
					Roles: []string{"master"},
					Env:   []string{"key=bar", "key=foo", "foo=bar", "IP=127.0.0.2"},
				},
			},
			SSH: v1.SSH{},
		},
	}
}

func Test_processor_WrapperShell(t *testing.T) {
	type fields struct {
		Cluster *v2.Cluster
	}
	type args struct {
		host  net.IP
		shell string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			"test command ENV",
			fields{Cluster: getTestCluster()},
			args{
				host:  net.ParseIP("192.168.0.2"),
				shell: "echo $foo ${IP[@]}",
			},
			"IP=127.0.0.2 key=(bar foo) foo=bar  && echo $foo ${IP[@]}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &processor{
				Cluster: tt.fields.Cluster,
			}
			if got := p.WrapperShell(tt.args.host, tt.args.shell); got != tt.want {
				t.Errorf("WrapperShell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processor_RenderAll(t *testing.T) {
	type fields struct {
		Cluster *v2.Cluster
	}
	type args struct {
		host net.IP
		dir  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"test render dir",
			fields{getTestCluster()},
			args{
				host: net.ParseIP("192.168.0.2"),
				dir:  "test/template",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &processor{
				Cluster: tt.fields.Cluster,
			}
			if err := p.RenderAll(tt.args.host, tt.args.dir); (err != nil) != tt.wantErr {
				t.Errorf("RenderAll() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
