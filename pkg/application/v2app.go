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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/pkg/define/application/v1"
	v12 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/rootfs"
	v2 "github.com/sealerio/sealer/types/api/v2"
	strUtils "github.com/sealerio/sealer/utils/strings"
	"github.com/sirupsen/logrus"
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
	extension v12.ImageExtension
}

func (a *v2Application) GetAppLaunchCmds(appName string) []string {
	return a.appLaunchCmdsMap[appName]
}

func (a *v2Application) GetAppNames() []string {
	return a.launchApps
}

func (a *v2Application) GetImageLaunchCmds() []string {
	if a.globalCmds != nil {
		return a.globalCmds
	}

	var cmds []string

	for _, appName := range a.launchApps {
		if appCmds, ok := a.appLaunchCmdsMap[appName]; ok {
			cmds = append(cmds, appCmds...)
		}
	}

	return cmds
}

func (a *v2Application) Launch(infraDriver infradriver.InfraDriver) error {
	var (
		rootfsPath = infraDriver.GetClusterRootfsPath()
		masters    = infraDriver.GetHostIPListByRole(common.MASTER)
		master0    = masters[0]
		launchCmds = a.GetImageLaunchCmds()
	)

	for _, cmdline := range launchCmds {
		if cmdline == "" {
			continue
		}

		if err := infraDriver.CmdAsync(master0, fmt.Sprintf(common.CdAndExecCmd, rootfsPath, cmdline)); err != nil {
			return err
		}
	}

	return nil
}

//Save application install history
//TODO save to cluster, also need a save struct.
func (a *v2Application) Save(opts SaveOptions) error {
	applicationFile := common.GetDefaultApplicationFile()

	f, err := os.OpenFile(filepath.Clean(applicationFile), os.O_RDWR|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return fmt.Errorf("cannot flock file %s - %s", applicationFile, err)
	}
	defer func() {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		if err != nil {
			logrus.Errorf("failed to unlock %s", applicationFile)
		}
	}()

	// TODO do not need all ImageExtension
	content, err := json.MarshalIndent(a.extension, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal image extension: %v", err)
	}

	if _, err = f.Write(content); err != nil {
		return err
	}

	return nil
}

func NewV2Application(app *v2.Application, extension v12.ImageExtension) (Interface, error) {
	v2App := &v2Application{
		app:              app,
		extension:        extension,
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
	appConfigMap := make(map[string]*v12.ApplicationConfig)
	for _, appConfig := range extension.Launch.AppConfigs {
		appConfigMap[appConfig.Name] = appConfig
	}
	for _, name := range v2App.launchApps {
		appRoot := makeItDir(filepath.Join(rootfs.GlobalManager.App().Root(), name))
		for _, exApp := range extension.Applications {
			v1app := exApp.(*v1.Application)
			if v1app.Name() != name {
				continue
			}
			if appConfig, ok := appConfigMap[name]; ok && appConfig.Launch != nil {
				v2App.appLaunchCmdsMap[name] = []string{v1app.LaunchCmd(appRoot, appConfig.Launch.CMDs)}
			} else {
				v2App.appLaunchCmdsMap[name] = []string{v1app.LaunchCmd(appRoot, nil)}
			}
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
