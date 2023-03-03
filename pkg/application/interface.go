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

import "github.com/sealerio/sealer/pkg/infradriver"

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
	// return spec.AppNames firstly Or get from image extension.
	GetAppNames() []string

	// GetDeleteCmds :get application delete commands from configs
	// return Delete.Cmds firstly Or wrapper application commands through its type.
	//GetDeleteCmds(appName string) []string

	// GetFileProcessors :get application file Processor using at mount stage to modify build app files.
	//GetFileProcessors(appName string) ([]FileProcessor, error)

	//GetImageName ()string
	//GetGlobalEnv() map[string]interface{}
	//AddGlobalEnv(envs []string)

	Launch(infraDriver infradriver.InfraDriver) error

	Save(opts SaveOptions) error
}

type FileProcessor interface {
	//Process application files though ValueType
	// currently Processor register as blew:
	// rawTypeProcessor: this will overwrite the FilePath with the Values.
	// argsTypeProcessor: this will render the FilePath with the Values.
	// secretTypeProcessor: this will write Values as Secret data to the file loaded from FilePath.
	// nameSpaceTypeProcessor: this will write Values as Namespace name to the FilePath whether it is exists or not.
	Process(appRoot string) error
}
