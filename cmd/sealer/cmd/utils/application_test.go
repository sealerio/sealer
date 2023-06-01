// Copyright © 2023 Alibaba Group Holding Ltd.
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

package utils

import (
	"testing"

	"github.com/sealerio/sealer/types/api/constants"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getNewApp() *v2.Application {
	newApp := &v2.Application{
		Spec: v2.ApplicationSpec{
			Cmds: []string{
				"cmd 1",
				"cmd 2",
			},
			LaunchApps: []string{
				"app1",
				"app2",
			},
			Configs: []v2.ApplicationConfig{
				{Env: []string{"Key=Value"}},
			},
		},
	}
	newApp.Name = "my-application"
	newApp.Kind = v2.GroupVersion.String()
	newApp.APIVersion = constants.ApplicationKind
	return newApp
}

func Test_ConstructApplication(t *testing.T) {
	expectedOverwriteCmds := &v2.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: constants.ApplicationKind,
			Kind:       v2.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-application",
		},
		Spec: v2.ApplicationSpec{
			Cmds: []string{
				"overwrite cmd 1",
				"overwrite cmd 2",
			},
			LaunchApps: []string{
				"app1",
				"app2",
			},
			Configs: []v2.ApplicationConfig{
				{Env: []string{"Key=Value"}},
			},
		},
	}

	expectedOverwriteAppNames := &v2.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: constants.ApplicationKind,
			Kind:       v2.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-application",
		},
		Spec: v2.ApplicationSpec{
			Cmds: []string{
				"cmd 1",
				"cmd 2",
			},
			LaunchApps: []string{
				"overwrite app1",
				"overwrite app2",
			},
			Configs: []v2.ApplicationConfig{
				{Env: []string{"Key=Value"}},
			},
		},
	}

	expectedOverwriteCmdsAndAppNames := &v2.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: constants.ApplicationKind,
			Kind:       v2.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-application",
		},
		Spec: v2.ApplicationSpec{
			Cmds: []string{
				"overwrite cmd 1",
				"overwrite cmd 2",
			},
			LaunchApps: []string{
				"overwrite app1",
				"overwrite app2",
			},
			Configs: []v2.ApplicationConfig{
				{Env: []string{"Key=Value"}},
			},
		},
	}

	expectedAddAppEnvs := &v2.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: constants.ApplicationKind,
			Kind:       v2.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-application",
		},
		Spec: v2.ApplicationSpec{
			Cmds: []string{
				"cmd 1",
				"cmd 2",
			},
			LaunchApps: []string{
				"app1",
				"app2",
			},
			Configs: []v2.ApplicationConfig{
				{Env: []string{"Key1=Value1", "Key=Value"}},
			},
		},
	}

	tests := []struct {
		name        string
		cmds        []string
		appNames    []string
		appEnvs     []string
		rawApp      *v2.Application
		expectedApp *v2.Application
	}{
		{
			name:        "test overwrite app cmds",
			cmds:        []string{"overwrite cmd 1", "overwrite cmd 2"},
			rawApp:      getNewApp(),
			expectedApp: expectedOverwriteCmds,
		},
		{
			name:        "test overwrite app names",
			appNames:    []string{"overwrite app1", "overwrite app2"},
			rawApp:      getNewApp(),
			expectedApp: expectedOverwriteAppNames,
		},
		{
			name:        "test overwrite app names and cmds",
			cmds:        []string{"overwrite cmd 1", "overwrite cmd 2"},
			appNames:    []string{"overwrite app1", "overwrite app2"},
			rawApp:      getNewApp(),
			expectedApp: expectedOverwriteCmdsAndAppNames,
		},
		{
			name:        "test add app envs",
			appEnvs:     []string{"Key1=Value1"},
			rawApp:      getNewApp(),
			expectedApp: expectedAddAppEnvs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedApp := ConstructApplication(tt.rawApp, tt.cmds, tt.appNames, tt.appEnvs)
			assert.Equal(t, expectedApp, tt.expectedApp)
		})
	}
}
