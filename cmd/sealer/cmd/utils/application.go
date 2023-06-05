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
	"github.com/sealerio/sealer/types/api/constants"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

// ConstructApplication merge flags to v2.Application
func ConstructApplication(app *v2.Application, cmds, appNames, globalEnvs []string) *v2.Application {
	var newApp *v2.Application

	if app != nil {
		newApp = app
	} else {
		newApp = &v2.Application{
			Spec: v2.ApplicationSpec{},
		}
		newApp.Name = "my-application"
		newApp.Kind = v2.GroupVersion.String()
		newApp.APIVersion = constants.ApplicationKind
	}

	if len(cmds) > 0 {
		newApp.Spec.Cmds = cmds
	}

	if appNames != nil {
		newApp.Spec.LaunchApps = appNames
	}

	// add appEnvs from flag to application object.
	if len(globalEnvs) > 0 {
		var appConfigList []v2.ApplicationConfig
		for _, appConfig := range newApp.Spec.Configs {
			appConfig.Env = append(globalEnvs, appConfig.Env...)
			appConfigList = append(appConfigList, appConfig)
		}
		newApp.Spec.Configs = appConfigList
	}

	return newApp
}
