// Copyright © 2021 Alibaba Group Holding Ltd.
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

package plugin

import "fmt"

var pluginFactories = make(map[string]Interface)

func Register(name string, factory Interface) {
	if factory == nil {
		panic("Must not provide nil pluginFactory")
	}
	_, registered := pluginFactories[name]
	if registered {
		panic(fmt.Sprintf("pluginFactory named %s already registered", name))
	}

	pluginFactories[name] = factory
}
