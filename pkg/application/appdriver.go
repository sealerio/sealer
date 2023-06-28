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
	imagev1 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/rootfs"
	v2 "github.com/sealerio/sealer/types/api/v2"
	mapUtils "github.com/sealerio/sealer/utils/maps"
	strUtils "github.com/sealerio/sealer/utils/strings"
	"github.com/sirupsen/logrus"
)

type applicationDriver struct {
	app *v2.Application

	// launchApps indicate that which applications will be launched
	//if launchApps==nil, use default launch apps got from image extension to launch.
	//if launchApps==[""], skip launch apps.
	//if launchApps==["app1","app2"], launch app1,app2.
	launchApps []string

	// registeredApps is the app name list which registered in image extension at build stage.
	registeredApps []string

	// globalCmds is raw cmds without any application info
	globalCmds []string

	// globalEnv is global env registered in image extension
	globalEnv map[string]string

	// appLaunchCmdsMap contains the whole appLaunchCmds with app name as its key.
	appLaunchCmdsMap map[string][]string
	//appDeleteCmdsMap    map[string][]string

	//extension is ImageExtension
	extension imagev1.ImageExtension

	// appRootMap contains the whole app root with app name as its key.
	appRootMap map[string]string

	// appEnvMap contains the whole app env with app name as its key.
	appEnvMap map[string]map[string]string

	// appFileProcessorMap contains the whole FileProcessors with app name as its key.
	appFileProcessorMap map[string][]FileProcessor
}

func (a *applicationDriver) GetAppLaunchCmds(appName string) []string {
	return a.appLaunchCmdsMap[appName]
}

func (a *applicationDriver) GetAppNames() []string {
	return a.launchApps
}

func (a *applicationDriver) GetAppRoot(appName string) string {
	return a.appRootMap[appName]
}

func (a *applicationDriver) GetImageLaunchCmds() []string {
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

func (a *applicationDriver) GetApplication() v2.Application {
	return *a.app
}

func (a *applicationDriver) Launch(infraDriver infradriver.InfraDriver) error {
	var (
		rootfsPath = infraDriver.GetClusterRootfsPath()
		masters    = infraDriver.GetHostIPListByRole(common.MASTER)
		master0    = masters[0]
		launchCmds = a.GetImageLaunchCmds()
	)

	logrus.Infof("start to launch sealer applications: %s", a.GetAppNames())

	logrus.Debugf("will to launch applications with cmd: %s", launchCmds)

	for _, cmdline := range launchCmds {
		if cmdline == "" {
			continue
		}

		if err := infraDriver.CmdAsync(master0, nil, fmt.Sprintf(common.CdAndExecCmd, rootfsPath, cmdline)); err != nil {
			return err
		}
	}

	return nil
}

// Save application install history
// TODO save to cluster, also need a save struct.
func (a *applicationDriver) Save(opts SaveOptions) error {
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

func (a *applicationDriver) FileProcess(mountDir string) error {
	for appName, processors := range a.appFileProcessorMap {
		for _, fp := range processors {
			if err := fp.Process(filepath.Join(mountDir, a.GetAppRoot(appName))); err != nil {
				return fmt.Errorf("failed to process appFiles for %s: %v", appName, err)
			}
		}
	}
	return nil
}

// NewAppDriver :unify v2.Application and image extension into same Interface using to do Application ops.
func NewAppDriver(app *v2.Application, extension imagev1.ImageExtension) (Interface, error) {
	appDriver := formatImageExtension(extension)
	appDriver.app = app
	// initialize globalCmds, overwrite default cmds from image extension.
	if len(app.Spec.Cmds) > 0 {
		appDriver.globalCmds = app.Spec.Cmds
	}

	// initialize appNames field, overwrite default app names from image extension.
	if app.Spec.LaunchApps != nil {
		// validate app.Spec.LaunchApps, if not in image extension,will return error
		// NOTE: app name =="" is valid
		for _, wanted := range app.Spec.LaunchApps {
			if len(wanted) == 0 {
				continue
			}
			if !strUtils.IsInSlice(wanted, appDriver.registeredApps) {
				return nil, fmt.Errorf("app name `%s` is not found in %s", wanted, appDriver.registeredApps)
			}
		}

		appDriver.launchApps = app.Spec.LaunchApps
	}

	// initialize Configs field
	for _, config := range app.Spec.Configs {
		if config.Name == "" {
			return nil, fmt.Errorf("application configs name could not be nil")
		}

		name := config.Name
		// make sure config in launchApps, if not will ignore this config.
		if !strUtils.IsInSlice(name, appDriver.launchApps) {
			continue
		}

		if config.Launch != nil {
			launchCmds := parseLaunchCmds(config.Launch)
			if launchCmds == nil {
				return nil, fmt.Errorf("failed to get launchCmds from application configs")
			}
			appDriver.appLaunchCmdsMap[name] = launchCmds
		}

		// merge config env with extension env
		if len(config.Env) > 0 {
			appEnvFromExtension := appDriver.appEnvMap[name]
			appEnvFromConfig := strUtils.ConvertStringSliceToMap(config.Env)
			appDriver.appEnvMap[name] = mapUtils.Merge(appEnvFromConfig, appEnvFromExtension)
		}

		// initialize app FileProcessors
		var fileProcessors []FileProcessor
		if len(appDriver.appEnvMap[name]) > 0 {
			fileProcessors = append(fileProcessors, envRender{envData: appDriver.appEnvMap[name]})
		}

		for _, appFile := range config.Files {
			fp, err := newFileProcessor(appFile)
			if err != nil {
				return nil, err
			}
			fileProcessors = append(fileProcessors, fp)
		}
		appDriver.appFileProcessorMap[name] = fileProcessors

		// TODO initialize delete field
	}

	return appDriver, nil
}

func formatImageExtension(extension imagev1.ImageExtension) *applicationDriver {
	appDriver := &applicationDriver{
		extension:           extension,
		globalCmds:          extension.Launch.Cmds,
		globalEnv:           extension.Env,
		launchApps:          extension.Launch.AppNames,
		registeredApps:      []string{},
		appLaunchCmdsMap:    map[string][]string{},
		appRootMap:          map[string]string{},
		appEnvMap:           map[string]map[string]string{},
		appFileProcessorMap: map[string][]FileProcessor{},
	}

	for _, registeredApp := range extension.Applications {
		appName := registeredApp.Name()
		// initialize app name
		appDriver.registeredApps = append(appDriver.registeredApps, appName)

		// initialize app root path
		appRoot := makeItDir(filepath.Join(rootfs.GlobalManager.App().Root(), appName))
		appDriver.appRootMap[appName] = appRoot

		// initialize app LaunchCmds
		app := registeredApp.(*v1.Application)
		appDriver.appLaunchCmdsMap[appName] = []string{v1.GetAppLaunchCmd(appRoot, app)}

		// initialize app env
		appDriver.appEnvMap[appName] = mapUtils.Merge(app.AppEnv, extension.Env)

		// initialize app FileProcessors
		if len(appDriver.appEnvMap[appName]) > 0 {
			appDriver.appFileProcessorMap[appName] = []FileProcessor{envRender{envData: appDriver.appEnvMap[appName]}}
		}
	}

	return appDriver
}

// parseLaunchCmds parse shell, kube,helm type launch cmds
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
