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
	v12 "github.com/sealerio/sealer/pkg/define/image/v1"
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

	extension := v12.ImageExtension{
		Launch: v12.Launch{
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

	driver, err := NewV2Application(app, extension)
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
