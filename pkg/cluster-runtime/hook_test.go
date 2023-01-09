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

package clusterruntime

//func TestLoadPluginsFromFile(t *testing.T) {
//	plugin1 := v1.Plugin{
//		Spec: v1.PluginSpec{
//			Type:   "SHELL",
//			Data:   "echo \"i am pre-init-host2 from rootfs\"",
//			Scope:  "master",
//			Action: "pre-init-host",
//		},
//	}
//	plugin1.Name = "pre-init-host2"
//	plugin1.Kind = "Plugin"
//	plugin1.APIVersion = "sealer.io/v1"
//
//	plugin2 := v1.Plugin{
//		Spec: v1.PluginSpec{
//			Type:   "SHELL",
//			Data:   "echo \"i am post-init-host2 from rootfs\"",
//			Scope:  "master",
//			Action: "post-init-host",
//		},
//	}
//	plugin2.Name = "post-init-host2"
//	plugin2.Kind = "Plugin"
//	plugin2.APIVersion = "sealer.io/v1"
//
//	type args struct {
//		data   string
//		wanted []v1.Plugin
//	}
//
//	var tests = []struct {
//		name string
//		args args
//	}{
//		{
//			"test load plugins from disk",
//			args{
//				data:   "./test",
//				wanted: []v1.Plugin{plugin1, plugin2},
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			result, err := LoadPluginsFromFile(tt.args.data)
//			if err != nil {
//				assert.Error(t, err)
//			}
//
//			assert.Equal(t, tt.args.wanted, result)
//		})
//	}
//
//}

//func TestTransferPluginsToHooks(t *testing.T) {
//	plugin1 := v1.Plugin{
//		Spec: v1.PluginSpec{
//			Type:   "SHELL",
//			Data:   "hostname",
//			Scope:  "master",
//			Action: "pre-init-host",
//		},
//	}
//	plugin1.Name = "MyHostname1"
//	plugin1.Kind = "Plugin"
//	plugin1.APIVersion = "sealer.io/v1"
//
//	plugin2 := v1.Plugin{
//		Spec: v1.PluginSpec{
//			Type:   "SHELL",
//			Data:   "kubectl get nodes",
//			Scope:  "master",
//			Action: "post-install",
//		},
//	}
//	plugin2.Name = "MyShell1"
//	plugin2.Kind = "Plugin"
//	plugin2.APIVersion = "sealer.io/v1"
//
//	plugin3 := v1.Plugin{
//		Spec: v1.PluginSpec{
//			Type:   "SHELL",
//			Data:   "hostname",
//			Scope:  "master",
//			Action: "pre-init-host",
//		},
//	}
//	plugin3.Name = "MyHostname2"
//	plugin3.Kind = "Plugin"
//	plugin3.APIVersion = "sealer.io/v1"
//
//	plugin4 := v1.Plugin{
//		Spec: v1.PluginSpec{
//			Type:   "SHELL",
//			Data:   "kubectl get pods",
//			Scope:  "node",
//			Action: "post-install",
//		},
//	}
//	plugin4.Name = "MyShell2"
//	plugin4.Kind = "Plugin"
//	plugin4.APIVersion = "sealer.io/v1"
//
//	wanted := map[Phase]HookConfigList{
//		Phase("pre-init-host"): []HookConfig{
//			{
//				Name:  "MyHostname1",
//				Type:  HookType("SHELL"),
//				Data:  "hostname",
//				Phase: Phase("pre-init-host"),
//				Scope: Scope("master"),
//			},
//			{
//				Name:  "MyHostname2",
//				Type:  HookType("SHELL"),
//				Data:  "hostname",
//				Phase: Phase("pre-init-host"),
//				Scope: Scope("node"),
//			},
//		},
//		Phase("post-install"): []HookConfig{
//			{
//				Name:  "MyShell1",
//				Type:  HookType("SHELL"),
//				Data:  "kubectl get nodes",
//				Phase: Phase("post-install"),
//				Scope: Scope("master"),
//			},
//			{
//				Name:  "MyShell2",
//				Type:  HookType("SHELL"),
//				Data:  "kubectl get pods",
//				Phase: Phase("post-install"),
//				Scope: Scope("master"),
//			},
//		},
//	}
//
//	type args struct {
//		data   []v1.Plugin
//		wanted map[Phase]HookConfigList
//	}
//
//	var tests = []struct {
//		name string
//		args args
//	}{
//		{
//			"test transfer plugins to hooks",
//			args{
//				data:   []v1.Plugin{plugin1, plugin2, plugin3, plugin4},
//				wanted: wanted,
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			result, err := transferPluginsToHooks(tt.args.data)
//			if err != nil {
//				assert.Error(t, err)
//			}
//
//			assert.Equal(t, tt.args.wanted, result)
//		})
//	}
//}
