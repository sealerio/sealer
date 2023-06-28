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

package v1

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sealerio/sealer/pkg/define/application"
	"github.com/sealerio/sealer/pkg/define/application/version"
)

type Application struct {
	NameVar    string   `json:"name"`
	TypeVar    string   `json:"type,omitempty"`
	FilesVar   []string `json:"files,omitempty"`
	VersionVar string   `json:"version,omitempty"`

	// AppEnv is a set of key value pair.
	// it is app level, only this app will be aware of its existence,
	// it is used to render app files, or as an environment variable for app startup and deletion commands
	AppEnv map[string]string `json:"env,omitempty"`

	// AppCMDs defined from `appcmds` instruction
	AppCMDs []string `json:"cmds,omitempty"`
}

func (app *Application) SetEnv(appEnv map[string]string) {
	app.AppEnv = appEnv
}

func (app *Application) SetCmds(appCmds []string) {
	app.AppCMDs = appCmds
}

func (app *Application) Version() string {
	return app.VersionVar
}

func (app *Application) Name() string {
	return app.NameVar
}

func (app *Application) Type() string {
	return app.TypeVar
}

func (app *Application) Files() []string {
	return app.FilesVar
}

// GetAppLaunchCmd : Get the real app launch cmds values in the following order.
// 1. appcmds instructionx defined in kubefile.
// 2. generated default command based on app type
func GetAppLaunchCmd(appRoot string, app *Application) string {
	if len(app.AppCMDs) != 0 {
		var cmds []string
		cmds = append(cmds, []string{"cd", appRoot}...)
		cmds = append(cmds, "&&")
		cmds = append(cmds, app.AppCMDs...)
		return strings.Join(cmds, " ")
	}
	switch app.Type() {
	case application.KubeApp:
		var cmds []string
		for _, file := range app.FilesVar {
			cmds = append(cmds, fmt.Sprintf("kubectl apply -f %s", filepath.Join(appRoot, file)))
		}
		return strings.Join(cmds, " && ")
	case application.HelmApp:
		return fmt.Sprintf("helm install %s %s", app.Name(), appRoot)
	case application.ShellApp:
		var cmds []string
		for _, file := range app.FilesVar {
			cmds = append(cmds, fmt.Sprintf("bash %s", filepath.Join(appRoot, file)))
		}
		return strings.Join(cmds, " && ")
	default:
		return ""
	}
}

func NewV1Application(
	name string,
	appType string, files []string) version.VersionedApplication {
	return &Application{
		NameVar:    name,
		TypeVar:    appType,
		FilesVar:   files,
		VersionVar: "v1",
	}
}
