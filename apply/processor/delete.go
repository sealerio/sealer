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

package processor

import (
	"fmt"

	"github.com/alibaba/sealer/utils"

	"github.com/alibaba/sealer/pkg/plugin"

	"github.com/alibaba/sealer/common"

	"github.com/alibaba/sealer/pkg/filesystem"
	"github.com/alibaba/sealer/pkg/runtime"
	v2 "github.com/alibaba/sealer/types/api/v2"
)

type DeleteProcessor struct {
	FileSystem filesystem.Interface
}

// Execute :according to the different of desired cluster to delete cluster.
func (d DeleteProcessor) Execute(cluster *v2.Cluster) (err error) {
	runTime, err := runtime.NewDefaultRuntime(cluster, cluster.GetAnnotationsByKey(common.ClusterfileName))
	if err != nil {
		return fmt.Errorf("failed to init runtime, %v", err)
	}

	err = runTime.Reset()
	if err != nil {
		return err
	}

	pipLine, err := d.GetPipeLine()
	if err != nil {
		return err
	}

	for _, f := range pipLine {
		if err = f(cluster); err != nil {
			return err
		}
	}

	return nil
}
func (d DeleteProcessor) GetPipeLine() ([]func(cluster *v2.Cluster) error, error) {
	var todoList []func(cluster *v2.Cluster) error
	todoList = append(todoList,
		d.ApplyCleanPlugin,
		d.UnMountRootfs,
		d.UnMountImage,
		d.CleanFS,
	)
	return todoList, nil
}

func (d DeleteProcessor) UnMountRootfs(cluster *v2.Cluster) error {
	hosts := append(cluster.GetMasterIPList(), cluster.GetNodeIPList()...)
	config := runtime.GetRegistryConfig(common.DefaultTheClusterRootfsDir(cluster.Name), runtime.GetMaster0Ip(cluster))
	if utils.NotIn(config.IP, hosts) {
		hosts = append(hosts, config.IP)
	}
	return d.FileSystem.UnMountRootfs(cluster, hosts)
}
func (d DeleteProcessor) UnMountImage(cluster *v2.Cluster) error {
	return d.FileSystem.UnMountImage(cluster)
}

func (d DeleteProcessor) ApplyCleanPlugin(cluster *v2.Cluster) error {
	plugins := plugin.NewPlugins(cluster.Name)
	if err := plugins.Dump(cluster.GetAnnotationsByKey(common.ClusterfileName)); err != nil {
		return err
	}
	if err := plugins.Load(); err != nil {
		return err
	}
	return plugins.Run(cluster, plugin.PhasePostClean)
}

func (d DeleteProcessor) CleanFS(cluster *v2.Cluster) error {
	return d.FileSystem.Clean(cluster)
}

func NewDeleteProcessor() (Interface, error) {
	fs, err := filesystem.NewFilesystem()
	if err != nil {
		return nil, err
	}

	return DeleteProcessor{
		FileSystem: fs,
	}, nil
}
