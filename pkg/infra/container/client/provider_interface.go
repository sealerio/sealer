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

package client

import (
	"github.com/docker/docker/api/types/mount"
)

type ProviderService interface {
	GetServerInfo() (*DockerInfo, error)
	RunContainer(opts *CreateOptsForContainer) (string, error)
	GetContainerInfo(containerID string, networkName string) (*Container, error)
	RmContainer(containerID string) error
	PullImage(imageName string) (string, error)
}

type Container struct {
	ContainerID       string
	NetworkID         string
	ContainerName     string
	ContainerHostName string
	ContainerIP       string
	Status            string
	ContainerLabel    map[string]string
}

type CreateOptsForContainer struct {
	ImageName         string
	NetworkName       string
	ContainerName     string
	ContainerHostName string
	ContainerLabel    map[string]string
	Mount             []mount.Mount
}

type DockerInfo struct {
	CgroupDriver    string
	CgroupVersion   string
	StorageDriver   string
	MemoryLimit     bool
	PidsLimit       bool
	CPUShares       bool
	CPUNumber       int
	SecurityOptions []string
}
