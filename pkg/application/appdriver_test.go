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

package application

import (
	"testing"

	v1 "github.com/sealerio/sealer/pkg/define/application/v1"
	"github.com/sealerio/sealer/pkg/define/application/version"
	imagev1 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/types/api/constants"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/stretchr/testify/assert"
)

func newTestApplication() (Interface, error) {
	app := &v2.Application{
		Spec: v2.ApplicationSpec{
			LaunchApps: []string{"nginx1", "nginx2"},
			Configs: []v2.ApplicationConfig{
				{
					Name: "nginx1",
					Launch: &v2.Launch{
						Cmds: []string{
							"kubectl apply -f ns.yaml",
							"kubectl apply -f nginx.yaml -n sealer-kube1",
						},
					},
				},
				{
					Name: "nginx2",
					Launch: &v2.Launch{
						Cmds: []string{
							"kubectl apply -f ns.yaml",
							"kubectl apply -f nginx.yaml -n sealer-kube2",
						},
					},
				},
			},
		},
	}
	app.Name = "my-app"
	app.Kind = constants.ApplicationKind
	app.APIVersion = v2.GroupVersion.String()

	extension := imagev1.ImageExtension{
		Launch: imagev1.Launch{
			AppNames: []string{"nginx1", "nginx2"},
		},
		Applications: []version.VersionedApplication{
			v1.NewV1Application(
				"nginx1",
				"kube",
				[]string{"nginx1.yaml"},
			),
			v1.NewV1Application(
				"nginx2",
				"kube",
				[]string{"nginx2.yaml"},
			),
			v1.NewV1Application(
				"nginx3",
				"kube",
				[]string{"nginx3.yaml"},
			),
		},
	}

	driver, err := NewAppDriver(app, extension)
	if err != nil {
		return nil, err
	}

	return driver, nil
}

func TestV2Application_GetLaunchCmds(t *testing.T) {
	driver, err := newTestApplication()
	if err != nil {
		assert.Error(t, err)
	}

	type args struct {
		driver  Interface
		appName string
		wanted  []string
	}
	var tests = []struct {
		name string
		args args
	}{
		{
			name: "get app launchCmds by name nginx1",
			args: args{
				driver:  driver,
				appName: "nginx1",
				wanted: []string{
					"kubectl apply -f ns.yaml",
					"kubectl apply -f nginx.yaml -n sealer-kube1",
				},
			},
		},
		{
			name: "get app launchCmds by name nginx2",
			args: args{
				driver:  driver,
				appName: "nginx2",
				wanted: []string{
					"kubectl apply -f ns.yaml",
					"kubectl apply -f nginx.yaml -n sealer-kube2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := driver.GetAppLaunchCmds(tt.args.appName)
			assert.Equal(t, tt.args.wanted, result)
		})
	}
}

func TestV2Application_GetAppNames(t *testing.T) {
	driver, err := newTestApplication()
	if err != nil {
		assert.Error(t, err)
	}

	type args struct {
		driver Interface
		wanted []string
	}
	var tests = []struct {
		name string
		args args
	}{{
		name: "get app launch names",
		args: args{
			driver: driver,
			wanted: []string{"nginx1", "nginx2"},
		},
	},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := driver.GetAppNames()
			assert.Equal(t, tt.args.wanted, result)
		})
	}
}

func TestV2Application_GetImageLaunchCmds(t *testing.T) {
	driver, err := newTestApplication()
	if err != nil {
		assert.Error(t, err)
	}

	type args struct {
		driver Interface
		wanted []string
	}
	var tests = []struct {
		name string
		args args
	}{{
		name: "get image launch cmds",
		args: args{
			driver: driver,
			wanted: []string{
				"kubectl apply -f ns.yaml",
				"kubectl apply -f nginx.yaml -n sealer-kube1",
				"kubectl apply -f ns.yaml",
				"kubectl apply -f nginx.yaml -n sealer-kube2",
			},
		},
	},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := driver.GetImageLaunchCmds()
			assert.Equal(t, tt.args.wanted, result)
		})
	}
}

func TestFormatExtensionToApplication(t *testing.T) {
	extension := imagev1.ImageExtension{
		Env: map[string]string{"globalKey": "globalValue"},
		Launch: imagev1.Launch{
			Cmds:     []string{"kubectl apply -f nginx.yaml"},
			AppNames: []string{"app1", "app2"},
		},
		Applications: []version.VersionedApplication{
			&v1.Application{
				NameVar:    "app1",
				TypeVar:    "kube",
				FilesVar:   []string{"/app1.yaml"},
				VersionVar: "v1",
				AppEnv:     map[string]string{"key1": "value1", "key2": "value2"},
				AppCMDs:    []string{"kubectl apply -f app1.yaml -n nginx-namespace"},
			},
			&v1.Application{
				NameVar:    "app2",
				TypeVar:    "kube",
				FilesVar:   []string{"/app2.yaml"},
				VersionVar: "v1",
				AppEnv:     map[string]string{"key3": "value3", "key4": "value4"},
				AppCMDs:    []string{"kubectl apply -f app2.yaml -n nginx-namespace"},
			},
			&v1.Application{
				NameVar:    "app3",
				TypeVar:    "kube",
				FilesVar:   []string{"/app3.yaml"},
				VersionVar: "v1",
				AppEnv:     map[string]string{"key5": "value5", "key6": "value6"},
				AppCMDs:    []string{"kubectl apply -f app3.yaml -n nginx-namespace"},
			},
		},
	}

	appEnvMap := map[string]map[string]string{
		"app1": {"key1": "value1", "key2": "value2", "globalKey": "globalValue"},
		"app2": {"key3": "value3", "key4": "value4", "globalKey": "globalValue"},
		"app3": {"key5": "value5", "key6": "value6", "globalKey": "globalValue"},
	}

	v2App := &applicationDriver{
		app:            nil,
		extension:      extension,
		globalCmds:     []string{"kubectl apply -f nginx.yaml"},
		globalEnv:      map[string]string{"globalKey": "globalValue"},
		launchApps:     []string{"app1", "app2"},
		registeredApps: []string{"app1", "app2", "app3"},
		appLaunchCmdsMap: map[string][]string{
			"app1": {"cd application/apps/app1/ && kubectl apply -f app1.yaml -n nginx-namespace"},
			"app2": {"cd application/apps/app2/ && kubectl apply -f app2.yaml -n nginx-namespace"},
			"app3": {"cd application/apps/app3/ && kubectl apply -f app3.yaml -n nginx-namespace"},
		},
		appRootMap: map[string]string{
			"app1": "application/apps/app1/",
			"app2": "application/apps/app2/",
			"app3": "application/apps/app3/",
		},
		appEnvMap: appEnvMap,
		appFileProcessorMap: map[string][]FileProcessor{
			"app1": {envRender{envData: appEnvMap["app1"]}},
			"app2": {envRender{envData: appEnvMap["app2"]}},
			"app3": {envRender{envData: appEnvMap["app3"]}},
		},
	}

	type args struct {
		input  imagev1.ImageExtension
		wanted *applicationDriver
	}
	var tests = []struct {
		name string
		args args
	}{{
		name: "format extension to application",
		args: args{
			input:  extension,
			wanted: v2App,
		},
	},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatImageExtension(tt.args.input)
			assert.Equal(t, tt.args.wanted, result)
		})
	}
}
