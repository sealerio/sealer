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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/imdario/mergo"
	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/pkg/define/application/v1"
	imagev1 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/rootfs"
	v2 "github.com/sealerio/sealer/types/api/v2"
	osUtils "github.com/sealerio/sealer/utils/os"
	strUtils "github.com/sealerio/sealer/utils/strings"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
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
	extension imagev1.ImageExtension

	// appRootMap contains the whole app root with app name as its key.
	appRootMap map[string]string

	// appFileProcessorMap contains the whole FileProcessors with app name as its key.
	appFileProcessorMap map[string][]FileProcessor
}

func (a *v2Application) GetAppLaunchCmds(appName string) []string {
	return a.appLaunchCmdsMap[appName]
}

func (a *v2Application) GetAppNames() []string {
	return a.launchApps
}

func (a *v2Application) GetAppRoot(appName string) string {
	return a.appRootMap[appName]
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

func (a *v2Application) FileProcess(mountDir string) error {
	for appName, processors := range a.appFileProcessorMap {
		for _, fp := range processors {
			if err := fp.Process(filepath.Join(mountDir, a.GetAppRoot(appName))); err != nil {
				return fmt.Errorf("failed to process appFiles for %s: %v", appName, err)
			}
		}
	}
	return nil
}

// NewV2Application :unify v2.Application and image extension into same Interface using to do Application ops.
func NewV2Application(app *v2.Application, extension imagev1.ImageExtension) (Interface, error) {
	v2App := &v2Application{
		app:                 app,
		globalCmds:          extension.Launch.Cmds,
		launchApps:          extension.Launch.AppNames,
		appLaunchCmdsMap:    map[string][]string{},
		appRootMap:          map[string]string{},
		appFileProcessorMap: map[string][]FileProcessor{},
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
	appConfigFromImageMap := make(map[string]*imagev1.ApplicationConfig)

	for _, appConfig := range extension.Launch.AppConfigs {
		appConfigFromImageMap[appConfig.Name] = appConfig
	}

	for _, name := range v2App.launchApps {
		appRoot := makeItDir(filepath.Join(rootfs.GlobalManager.App().Root(), name))
		v2App.appRootMap[name] = appRoot
		for _, exApp := range extension.Applications {
			v1app := exApp.(*v1.Application)
			if v1app.Name() != name {
				continue
			}
			if appConfig, ok := appConfigFromImageMap[name]; ok && appConfig.Launch != nil {
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

		// initialize app files
		var fileProcessors []FileProcessor
		for _, appFile := range config.Files {
			fp, err := newFileProcessor(appFile)
			if err != nil {
				return nil, err
			}
			fileProcessors = append(fileProcessors, fp)
		}
		v2App.appFileProcessorMap[name] = fileProcessors

		// TODO initialize delete field
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

func newFileProcessor(appFile v2.AppFile) (FileProcessor, error) {
	switch appFile.Strategy {
	case v2.OverWriteStrategy:
		return overWriteProcessor{appFile}, nil
	case v2.MergeStrategy:
		return mergeProcessor{appFile}, nil
	}

	return nil, fmt.Errorf("failed to init fileProcessor,%s is not register", appFile.Strategy)
}

// overWriteProcessor :this will overwrite the FilePath with the Values.
type overWriteProcessor struct {
	v2.AppFile
}

func (r overWriteProcessor) Process(appRoot string) error {
	target := filepath.Join(appRoot, r.Path)

	err := osUtils.NewCommonWriter(target).WriteFile([]byte(r.Data))
	if err != nil {
		return fmt.Errorf("failed to write to file %s with raw mode: %v", target, err)
	}
	return nil
}

// mergeProcessor :this will merge the FilePath with the Values.
//Only files in yaml format are supported.
//if Strategy is "merge" will deeply merge each yaml file section.
type mergeProcessor struct {
	v2.AppFile
}

func (m mergeProcessor) Process(appRoot string) error {
	var (
		result     [][]byte
		srcDataMap = make(map[string]interface{})
	)

	err := yaml.Unmarshal([]byte(m.Data), &srcDataMap)
	if err != nil {
		return fmt.Errorf("failed to load config data: %v", err)
	}

	target := filepath.Join(appRoot, m.Path)
	contents, err := os.ReadFile(filepath.Clean(target))
	if err != nil {
		return err
	}

	for _, section := range bytes.Split(contents, []byte("---\n")) {
		destDataMap := make(map[string]interface{})

		err = yaml.Unmarshal(section, &destDataMap)
		if err != nil {
			return fmt.Errorf("failed to unmarshal config data: %v", err)
		}

		err = mergo.Merge(&destDataMap, &srcDataMap, mergo.WithOverride)
		if err != nil {
			return fmt.Errorf("failed to merge config: %v", err)
		}

		out, err := yaml.Marshal(destDataMap)
		if err != nil {
			return err
		}

		result = append(result, out)
	}

	err = osUtils.NewCommonWriter(target).WriteFile(bytes.Join(result, []byte("---\n")))
	if err != nil {
		return fmt.Errorf("failed to write to file %s with raw mode: %v", target, err)
	}
	return nil
}

// renderProcessor : this will render the FilePath with the Values.
//type renderProcessor struct {
//	v2.AppFile
//}

//const templateSuffix = ".tmpl"

//func (a renderProcessor) Process(appRoot string) error {
//	target := filepath.Join(appRoot, a.Path)
//
//	if !strings.HasSuffix(a.Path, templateSuffix) {
//		return nil
//	}
//
//	writer, err := os.OpenFile(filepath.Clean(strings.TrimSuffix(target, templateSuffix)), os.O_CREATE|os.O_RDWR, os.ModePerm)
//	if err != nil {
//		return fmt.Errorf("failed to open file [%s] when render args: %v", target, err)
//	}
//
//	defer func() {
//		_ = writer.Close()
//	}()
//
//	t, err := template.New(a.Path).ParseFiles(target)
//	if err != nil {
//		return fmt.Errorf("failed to create template(%s): %v", target, err)
//	}
//
//	if err := t.Execute(writer, strUtils.ConvertEnv(strings.Split(a.Data, " "))); err != nil {
//		return fmt.Errorf("failed to render file %s with args mode: %v", target, err)
//	}
//
//	return nil
//}
