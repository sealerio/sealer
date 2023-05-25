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
	"github.com/sealerio/sealer/pkg/infradriver"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

// Interface works like application driver,
// it converts Application fields, such as app configs, app global envs, app image names and so on.
type Interface interface {
	// GetImageLaunchCmds :its image level. get entire application launch commands
	// return appended each app launch cmds Or globalCmds.
	GetImageLaunchCmds() []string

	// GetAppLaunchCmds :get application launch commands from configs
	// return Launch.Cmds firstly Or wrapper application commands through its type.
	GetAppLaunchCmds(appName string) []string

	// GetAppNames :get application name list
	// return spec.AppNames
	GetAppNames() []string

	//GetAppRoot :get appRoot path by its name.
	GetAppRoot(appName string) string

	// GetDeleteCmds :get application delete commands from configs
	// return Delete.Cmds firstly Or wrapper application commands through its type.
	//GetDeleteCmds(appName string) []string

	// FileProcess :Process application file using at mount stage to modify build app files.
	FileProcess(mountDir string) error

	// GetApplication :get application spec
	// return v2.Application
	GetApplication() v2.Application

	//GetImageName ()string
	//GetGlobalEnv() map[string]interface{}
	//AddGlobalEnv(envs []string)

	Launch(infraDriver infradriver.InfraDriver) error

	Save(opts SaveOptions) error
}

type FileProcessor interface {
	//Process application files though ValueType
	// currently Processor register as blew:
	// overWriteProcessor: this will overwrite the FilePath with the Values.
	// renderProcessor: this will render the FilePath with the Values.
	Process(appRoot string) error
}
