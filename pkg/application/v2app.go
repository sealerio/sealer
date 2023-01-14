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
	"fmt"
	"path/filepath"
	"strings"

	v1 "github.com/sealerio/sealer/pkg/define/application/v1"
	v12 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/rootfs"
	v2 "github.com/sealerio/sealer/types/api/v2"
	strUtils "github.com/sealerio/sealer/utils/strings"
)

type v2Application struct {
	app *v2.Application

	// launchApps indicate that which applications will be launched
	launchApps []string

	// globalCmds is raw cmds without any application info
	globalCmds []string

	// appLaunchCmdsMap contains the whole appLaunchCmds with app name as its key.
	appLaunchCmdsMap map[string][]string
	//appDeleteCmdsMap    map[string][]string
	//appFileProcessorMap map[string][]Processor
}

func (a *v2Application) GetAppLaunchCmds(appName string) []string {
	return a.appLaunchCmdsMap[appName]
}

func (a *v2Application) GetAppNames() []string {
	return a.launchApps
}

func (a *v2Application) GetGlobalCmds() []string {
	return a.globalCmds
}

func (a *v2Application) GetImageLaunchCmds() []string {
	if a.globalCmds != nil {
		return a.globalCmds
	}

	var cmds []string
	for _, v := range a.appLaunchCmdsMap {
		cmds = append(cmds, v...)
	}

	return cmds
}

func NewV2Application(app *v2.Application, extension v12.ImageExtension) (Interface, error) {
	v2App := &v2Application{
		app:              app,
		globalCmds:       extension.Launch.Cmds,
		launchApps:       extension.Launch.AppNames,
		appLaunchCmdsMap: map[string][]string{},
	}

	// initialize globalCmds, overwrite default cmds from image extension.
	if len(app.Spec.Cmds) > 0 {
		v2App.globalCmds = app.Spec.Cmds
	}

	// initialize appNames field, overwrite default app names from image extension.
	if len(app.Spec.LaunchApps) > 0 {
		v2App.launchApps = app.Spec.LaunchApps
	}

	// initialize appLaunchCmdsMap, get default launch cmds from image extension.
	for _, name := range v2App.launchApps {
		appRoot := makeItDir(filepath.Join(rootfs.GlobalManager.App().Root(), name))
		for _, exApp := range extension.Applications {
			v1app := exApp.(*v1.Application)
			if v1app.Name() != name {
				continue
			}
			v2App.appLaunchCmdsMap[name] = []string{v1app.LaunchCmd(appRoot)}
		}
	}

	// initialize Configs field
	for _, config := range app.Spec.Configs {
		if config.Name == "" {
			return nil, fmt.Errorf("v2Application configs name coule not be nil")
		}

		name := config.Name
		// make sure config in launchApps,if not will ignore this config.
		if !strUtils.IsInSlice(name, v2App.launchApps) {
			continue
		}

		if config.Launch != nil {
			launchCmds := parseLaunchCmds(config.Launch)
			if launchCmds == nil {
				return nil, fmt.Errorf("failed to get launchCmds from v2Application configs")
			}

			v2App.appLaunchCmdsMap[name] = launchCmds
		}
		// TODO initialize delete field
		// TODO initialize files field
	}

	return v2App, nil
}

//parseLaunchCmds parse shell, kube,helm type launch cmds
// kubectl apply -n sealer-io -f ns.yaml -f app.yaml
// helm install my-nginx bitnami/nginx
// key1=value1 key2=value2 && bash install1.sh && bash install2.sh
func parseLaunchCmds(launch *v2.Launch) []string {
	if launch.Cmds != nil {
		return launch.Cmds
	}
	// TODO add shell,helm,kube type cmds.
	return nil
}

func makeItDir(str string) string {
	if !strings.HasSuffix(str, "/") {
		return str + "/"
	}
	return str
}
