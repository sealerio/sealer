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

import "github.com/sealerio/sealer/pkg/rootfs/define"

type manager struct {
	app *application
}

type application struct {
	appRootRelPath string
}

func (a *application) Root() string {
	return a.appRootRelPath
}

func (m *manager) App() define.App {
	return m.app
}

func NewManager() define.Manager {
	return &manager{
		app: &application{appRootRelPath: "application/apps/"},
	}
}
