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
	"github.com/sealerio/sealer/pkg/define/application/version"
)

type Application struct {
	NameVar    string `json:"name"`
	TypeVar    string `json:"type,omitempty"`
	VersionVar string `json:"version,omitempty"`
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

func NewV1Application(
	name string,
	appType string) version.VersionedApplication {
	return &Application{
		NameVar:    name,
		TypeVar:    appType,
		VersionVar: "v1",
	}
}
