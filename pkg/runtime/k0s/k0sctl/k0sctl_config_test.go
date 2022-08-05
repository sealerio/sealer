// Copyright © 2022 Alibaba Group Holding Ltd.
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

package k0sctl

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/k0sproject/dig"
	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	"github.com/k0sproject/rig"
	"github.com/sealerio/sealer/pkg/runtime/k0s/k0sctl/v1beta1"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var mockClusterFile = v2.Cluster{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "sealer.cloud/v2",
		Kind:       "Cluster",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "my-k0s-cluster",
	},
	Spec: v2.ClusterSpec{
		Image: "k0s:v1.24.1-rc",
		SSH: v1.SSH{
			Passwd: "test123",
			Port:   "2222",
		},
		Hosts: []v2.Host{
			{
				IPS:   []net.IP{net.IP("192.168.0.2")},
				Roles: []string{"master"},
				SSH: v1.SSH{
					Passwd: "yyy",
					Port:   "22",
				},
			},
			{
				IPS:   []net.IP{net.IP("192.168.0.3"), net.IP("192.168.0.4")},
				Roles: []string{"worker"},
			},
			{
				IPS:   []net.IP{net.IP("192.168.0.5")},
				Roles: []string{"worker"},
			},
		},
	},
}

var mockK0sctlConfig = v1beta1.Cluster{
	APIVersion: "k0sctl.k0sproject.io/v1beta1",
	Kind:       "Cluster",
	Metadata: &v1beta1.ClusterMetadata{
		Name: "my-k0s-cluster",
	},
	Spec: &cluster.Spec{
		Hosts: cluster.Hosts{
			{
				Role:         "controller",
				InstallFlags: []string{"--debug"},
				Connection: rig.Connection{
					SSH: &rig.SSH{
						Address: "172.16.161.64",
						User:    "root",
						Port:    22,
						KeyPath: "~/.ssh/id_rsa",
					},
				},
				UploadBinary:  true,
				K0sBinaryPath: K0sUploadBinaryPath,
			},
			{
				Role:         "worker",
				InstallFlags: []string{"--debug"},
				Connection: rig.Connection{
					SSH: &rig.SSH{
						Address: "172.16.161.65",
						User:    "root",
						Port:    22,
						KeyPath: "~/.ssh/id_rsa",
					},
				},
				UploadBinary:  true,
				K0sBinaryPath: K0sUploadBinaryPath,
			},
		},
		K0s: &cluster.K0s{
			Version: "v1.24.2+k0s.0",
			Config: dig.Mapping{
				"apiVersion": "k0s.k0sproject.io/v1beta1",
				"spec": dig.Mapping{
					"images": dig.Mapping{
						//"repository": "sea.hub:5000",
					},
				},
			},
		},
	},
}

func TestK0sConfig_parseSSHPortByIP(t *testing.T) {
	type fields struct {
		Cluster *v1beta1.Cluster
	}
	type args struct {
		ipAddr string
		hosts  []v2.Host
		ssh    v1.SSH
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name:   "test ip overwrite",
			fields: fields{Cluster: &v1beta1.Cluster{}},
			args: args{
				ipAddr: "192.168.0.2",
				hosts: []v2.Host{
					{
						IPS:   []net.IP{net.ParseIP("192.168.0.2")},
						Roles: []string{"master"},
						SSH: v1.SSH{
							Passwd: "yyy",
							Port:   "22",
						},
					},
					{
						IPS:   []net.IP{net.ParseIP("192.168.0.3"), net.ParseIP("192.168.0.4")},
						Roles: []string{"worker"},
					},
					{
						IPS:   []net.IP{net.ParseIP("192.168.0.5")},
						Roles: []string{"worker"},
					},
				},
				ssh: v1.SSH{
					Passwd: "test123",
					Port:   "2222",
				},
			},
			want:    22,
			wantErr: false,
		},
		{
			name:   "test ip global cover",
			fields: fields{Cluster: &v1beta1.Cluster{}},
			args: args{
				ipAddr: "192.168.0.3",
				hosts: []v2.Host{
					{
						IPS:   []net.IP{net.ParseIP("192.168.0.2")},
						Roles: []string{"master"},
						SSH: v1.SSH{
							Passwd: "yyy",
							Port:   "22",
						},
					},
					{
						IPS:   []net.IP{net.ParseIP("192.168.0.3"), net.ParseIP("192.168.0.4")},
						Roles: []string{"worker"},
					},
					{
						IPS:   []net.IP{net.ParseIP("192.168.0.5")},
						Roles: []string{"worker"},
					},
				},
				ssh: v1.SSH{
					Passwd: "test123",
					Port:   "2222",
				},
			},
			want:    2222,
			wantErr: false,
		},
		{
			name:   "test incorrect with port parse error",
			fields: fields{Cluster: &v1beta1.Cluster{}},
			args: args{
				ipAddr: "192.168.0.2",
				hosts: []v2.Host{
					{
						IPS:   []net.IP{net.ParseIP("192.168.0.2")},
						Roles: []string{"master"},
						SSH: v1.SSH{
							Passwd: "yyy",
							Port:   "error",
						},
					},
					{
						IPS:   []net.IP{net.ParseIP("192.168.0.3"), net.ParseIP("192.168.0.4")},
						Roles: []string{"worker"},
					},
					{
						IPS:   []net.IP{net.ParseIP("192.168.0.5")},
						Roles: []string{"worker"},
					},
				},
				ssh: v1.SSH{
					Passwd: "test123",
					Port:   "2222",
				},
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &K0sConfig{
				Cluster: tt.fields.Cluster,
			}
			got, err := c.parseSSHPortByIP(tt.args.ipAddr, tt.args.hosts, tt.args.ssh)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSSHPortByIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseSSHPortByIP() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestK0sConfig_addHostField(t *testing.T) {
	type fields struct {
		Cluster *v1beta1.Cluster
	}
	type args struct {
		ipsAddr string
		port    int
		role    string
		user    string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "test add Node",
			fields: fields{Cluster: &mockK0sctlConfig},
			args: args{
				ipsAddr: "10.0.0.2",
				port:    22,
				role:    "worker",
				user:    "root",
			},
		},
		{
			name:   "test add Master",
			fields: fields{Cluster: &mockK0sctlConfig},
			args: args{
				ipsAddr: "10.0.0.4",
				port:    22,
				role:    "controller",
				user:    "root",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &K0sConfig{
				Cluster: tt.fields.Cluster,
			}
			c.addHostField(tt.args.ipsAddr, tt.args.port, tt.args.role, tt.args.user)
			fmt.Println(c.Cluster.Spec.Hosts)
			if err := c.Validate(); err != nil {
				t.Errorf("add field wrong: %v", err)
			}
		})
	}
}

func TestK0sConfig_DefineConfigFork0s(t *testing.T) {
	type fields struct {
		Cluster *v1beta1.Cluster
	}
	type args struct {
		version string
		domain  string
		port    string
		name    string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "test set repo field",
			fields: fields{
				Cluster: &mockK0sctlConfig,
			},
			args: args{
				version: "v1.24.2+k0s.0",
				domain:  "sea.hub",
				port:    "5000",
				name:    "my-k0s-cluster",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &K0sConfig{
				Cluster: tt.fields.Cluster,
			}
			c.DefineConfigFork0s(tt.args.version, tt.args.domain, tt.args.port, tt.args.name)
			fmt.Println(c.Spec.K0s.Config.Dig("spec", "images", "repository"))
			if err := c.Validate(); err != nil {
				t.Errorf("add config error: %v", err)
			}
		})
	}
}

func TestK0sConfig_convertIPVSToAddress(t *testing.T) {
	type fields struct {
		Cluster *v1beta1.Cluster
	}
	type args struct {
		clusterFile *v2.Cluster
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "cluster file to k0s config",
			fields: fields{
				Cluster: &mockK0sctlConfig,
			},
			args: args{
				clusterFile: &mockClusterFile,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &K0sConfig{
				Cluster: tt.fields.Cluster,
			}
			if err := c.convertIPVSToAddress(tt.args.clusterFile); (err != nil) != tt.wantErr {
				t.Errorf("convertIPVSToAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestK0sConfig_WriteConfigToMaster0(t *testing.T) {
	type fields struct {
		Cluster *v1beta1.Cluster
	}
	type args struct {
		rootfs string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test write rootfs/k0sctl.yaml",
			fields: fields{
				Cluster: &mockK0sctlConfig,
			},
			args: args{
				rootfs: "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//prevent the read permission denied.
			oldmask := syscall.Umask(0)
			defer syscall.Umask(oldmask)

			dir, err := os.MkdirTemp("", "test-rootfs-metadata-tmp")
			if err != nil {
				t.Errorf("Make temp dir %s error = %s, wantErr %v", dir, err, tt.wantErr)
			}
			defer func() {
				err = os.RemoveAll(dir)
				if err != nil {
					t.Errorf("Remove temp dir %s error = %v, wantErr %v", dir, err, tt.wantErr)
				}
			}()
			tt.args.rootfs = dir
			fmt.Println(tt.args.rootfs)
			c := &K0sConfig{
				Cluster: tt.fields.Cluster,
			}
			if err := c.WriteConfigToMaster0(tt.args.rootfs); (err != nil) != tt.wantErr {
				t.Errorf("WriteConfigToMaster0() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
