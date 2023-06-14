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

package clusterfile

import (
	"net"
	"os"
	"testing"

	"github.com/sealerio/sealer/types/api/constants"

	"github.com/stretchr/testify/assert"
	"k8s.io/kube-proxy/config/v1alpha1"

	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

const data = `apiVersion: sealer.com/v1alpha1
kind: Config
metadata:
  name: mysql-config
spec:
  path: etc/mysql.yaml
  data: |
       mysql-user: root
       mysql-passwd: xxx
---
apiVersion: sealer.io/v2
kind: Application
metadata:
  name: my-apps
spec:
  launchApps:
    - app1
    - app2
  configs:
    - name: app2
      launch:
        cmds:
          - kubectl get pods -A
---
apiVersion: sealer.io/v1
kind: Plugin
metadata:
  name: MyHostname # Specify this plugin name,will dump in $rootfs/plugins dir.
spec:
  type: HOSTNAME # fixed string,should not change this name.
  action: PreInit # Specify which phase to run.
  data: |
    192.168.0.2 master-0
---
apiVersion: sealer.io/v1
kind: Plugin
metadata:
  name: MyShell # Specify this plugin name,will dump in $rootfs/plugins dir.
spec:
  type: SHELL
  action: PostInstall # PreInit PostInstall
  scope: master
  data: |
    kubectl get nodes
---
apiVersion: sealer.io/v2
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: kubernetes:v1.19.8
  env:
    - key1=value1
    - key2=value2;value3 #key2=[value2, value3]
  ssh:
    passwd: test123
    pk: xxx
    pkPasswd: xxx
    user: root
    port: "22"
  hosts:
    - ips: [ 192.168.0.2 ]
      roles: [ master ] # add role field to specify the node role
      env: # rewrite some nodes has different env config
        - etcd-dir=/data/etcd
      ssh: # rewrite ssh config if some node has different passwd...
        user: root
        passwd: test456
        port: "22"
    - ips: [ 192.168.0.3 ]
      roles: [ node,db ]
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
localAPIEndpoint:
  bindPort: 6443
nodeRegistration:
  criSocket: /var/run/dockershim.sock
---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: "ipvs"
ipvs:
  excludeCIDRs:
    - "10.103.97.2/32"`

func TestDecodeClusterFile(t *testing.T) {
	cluster := v2.Cluster{
		Spec: v2.ClusterSpec{
			Image:    "kubernetes:v1.19.8",
			DataRoot: "/var/lib/sealer/data",
			Env:      []string{"key1=value1", "key2=value2;value3", "LocalRegistryDomain=sea.hub", "LocalRegistryPort=5000", "LocalRegistryURL=sea.hub:5000", "RegistryDomain=sea.hub", "RegistryPort=5000", "RegistryURL=sea.hub:5000"},
			SSH: v1.SSH{
				User:     "root",
				Passwd:   "test123",
				Port:     "22",
				Pk:       "xxx",
				PkPasswd: "xxx",
			},
			Registry: v2.Registry{
				LocalRegistry: &v2.LocalRegistry{
					RegistryConfig: v2.RegistryConfig{
						Domain: "sea.hub",
						Port:   5000,
					},
					HA:       &defaultHA,
					Insecure: &defaultInsecure,
				},
			},
			Hosts: []v2.Host{
				{
					IPS:   []net.IP{net.IPv4(192, 168, 0, 2)},
					Roles: []string{"master"},
					Env:   []string{"etcd-dir=/data/etcd"},
					SSH: v1.SSH{
						User:   "root",
						Passwd: "test456",
						Port:   "22",
					},
				},
				{
					IPS:   []net.IP{net.IPv4(192, 168, 0, 3)},
					Roles: []string{"node", "db"},
				},
			},
		},
	}
	cluster.APIVersion = "sealer.io/v2"
	cluster.Kind = "Cluster"
	cluster.Name = "my-cluster"

	plugin1 := v1.Plugin{
		Spec: v1.PluginSpec{
			Type:   "HOSTNAME",
			Data:   "192.168.0.2 master-0\n",
			Action: "PreInit",
		},
	}
	plugin1.Name = "MyHostname"
	plugin1.Kind = constants.PluginKind
	plugin1.APIVersion = v1.GroupVersion.String()

	plugin2 := v1.Plugin{
		Spec: v1.PluginSpec{
			Type:   "SHELL",
			Data:   "kubectl get nodes\n",
			Scope:  "master",
			Action: "PostInstall",
		},
	}
	plugin2.Name = "MyShell"
	plugin2.Kind = "Plugin"
	plugin2.APIVersion = "sealer.io/v1"

	config := v1.Config{
		Spec: v1.ConfigSpec{
			Path: "etc/mysql.yaml",
			Data: "mysql-user: root\nmysql-passwd: xxx\n",
		},
	}
	config.Name = "mysql-config"
	config.Kind = "Config"
	config.APIVersion = "sealer.com/v1alpha1"

	app := &v2.Application{
		Spec: v2.ApplicationSpec{
			LaunchApps: []string{"app1", "app2"},
			Configs: []v2.ApplicationConfig{
				{
					Name: "app2",
					Launch: &v2.Launch{
						Cmds: []string{
							"kubectl get pods -A",
						},
					},
				},
			},
		},
	}
	app.Name = "my-apps"
	app.Kind = constants.ApplicationKind
	app.APIVersion = v2.GroupVersion.String()

	type wanted struct {
		cluster     v2.Cluster
		config      []v1.Config
		plugins     []v1.Plugin
		application *v2.Application
	}

	type args struct {
		data   []byte
		wanted wanted
	}

	var tests = []struct {
		name string
		args args
	}{
		{
			"test decode cluster file",
			args{
				data: []byte(data),
				wanted: wanted{
					cluster:     cluster,
					config:      []v1.Config{config},
					plugins:     []v1.Plugin{plugin1, plugin2},
					application: app,
				},
			},
		},
	}

	f, err := os.CreateTemp("", "tmpfile-")
	if err != nil {
		assert.Error(t, err)
	}

	defer os.Remove(f.Name())

	if _, err = f.Write([]byte(data)); err != nil {
		assert.Error(t, err)
	}
	if err = f.Close(); err != nil {
		assert.Error(t, err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := NewClusterFile([]byte(data))
			if err != nil {
				assert.Errorf(t, err, "failed to NewClusterFile by name")
			}

			assert.NotNil(t, i)

			assert.Equal(t, tt.args.wanted.cluster, i.GetCluster())

			assert.Equal(t, tt.args.wanted.config, i.GetConfigs())

			assert.Equal(t, tt.args.wanted.plugins, i.GetPlugins())

			assert.Equal(t, tt.args.wanted.application, i.GetApplication())

			kubeadm := i.GetKubeadmConfig()
			assert.NotNil(t, kubeadm)

			assert.Equal(t, kubeadm.LocalAPIEndpoint.BindPort, int32(6443))
			assert.Equal(t, kubeadm.InitConfiguration.NodeRegistration.CRISocket, "/var/run/dockershim.sock")

			assert.Equal(t, kubeadm.Mode, v1alpha1.ProxyMode("ipvs"))
			assert.Equal(t, kubeadm.IPVS.ExcludeCIDRs, []string{"10.103.97.2/32"})
		})
	}
}
