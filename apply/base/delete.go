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

package base

import (
	"fmt"

	"github.com/alibaba/sealer/filesystem"
	"github.com/alibaba/sealer/runtime"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type DeleteApply struct {
	FileSystem filesystem.Interface
}

// DoApply do apply: do truly apply,input is desired cluster .
func (d DeleteApply) DoApply(cluster *v1.Cluster) (err error) {
	runTime, err := runtime.NewDefaultRuntime(cluster)
	if err != nil {
		return fmt.Errorf("failed to init runtime, %v", err)
	}

	err = runTime.Reset(cluster)
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
func (d DeleteApply) GetPipeLine() ([]func(cluster *v1.Cluster) error, error) {
	var todoList []func(cluster *v1.Cluster) error
	todoList = append(todoList,
		d.UnMountRootfs,
		d.UnMountImage,
		d.CleanFS,
	)
	return todoList, nil
}

func (d DeleteApply) UnMountRootfs(cluster *v1.Cluster) error {
	return d.FileSystem.UnMountRootfs(cluster)
}
func (d DeleteApply) UnMountImage(cluster *v1.Cluster) error {
	return d.FileSystem.UnMountImage(cluster)
}

func (d DeleteApply) CleanFS(cluster *v1.Cluster) error {
	return d.FileSystem.Clean(cluster)
}

func NewDeleteApply() (Interface, error) {
	fs, err := filesystem.NewFilesystem()
	if err != nil {
		return nil, err
	}

	return DeleteApply{
		FileSystem: fs,
	}, nil
}
