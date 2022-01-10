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

package docker

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

func (d Docker) RmContainerByName(containerName string) error {
	containers, err := d.GetContainerListByName(containerName)
	if err != nil {
		return err
	}
	for _, c := range containers {
		err = d.cli.ContainerRemove(d.ctx, c.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (d Docker) GetContainerListByName(containerName string) ([]types.Container, error) {
	opts := types.ContainerListOptions{All: true}
	opts.Filters = filters.NewArgs()
	opts.Filters.Add("name", containerName)
	containers, err := d.cli.ContainerList(d.ctx, opts)

	if err != nil {
		return nil, err
	}

	return containers, err
}
