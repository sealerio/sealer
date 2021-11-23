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

import (
	"fmt"
	"github.com/alibaba/sealer/common"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"path/filepath"
	"plugin"
)

// RunPlugin out of tree plugins need to implement this interface.
type RunPlugin interface {
	Exec(cluster *v1.Cluster) error
}

type Golang struct{}

func NewGolangPlugin() Interface {
	return &Golang{}
}

func (g Golang) Run(context Context, phase Phase) error {
	if string(phase) != context.Plugin.Spec.Action || context.Plugin.Spec.Type != GolangPlugin {
		return nil
	}
	plug, err := plugin.Open(filepath.Join(common.DefaultTheClusterRootfsPluginDir(context.Cluster.Name), context.Plugin.Name))
	if err != nil {
		return err
	}
	//look up the exposed variable named `Plugin`
	symbol, err := plug.Lookup(Plugin)
	if err != nil {
		return err
	}

	p, ok := symbol.(RunPlugin)
	if !ok {
		return fmt.Errorf("failed to find GOLANG plugin symbol %s", context.Plugin.Name)
	}

	return p.Exec(context.Cluster)
}
