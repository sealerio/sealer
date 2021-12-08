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
	"context"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

type Provider struct {
	DockerClient *client.Client
	Ctx          context.Context
}

type Container struct {
	ContainerID       string
	ContainerName     string
	ContainerHostName string
	ContainerIP       string
	Status            string
	ContainerLabel    map[string]string
}

type CreateOptsForContainer struct {
	ImageName         string
	NetworkId         string
	NetworkName       string
	ContainerName     string
	ContainerHostName string
	ContainerLabel    map[string]string
	Mount             *mount.Mount
	IsMaster0         bool
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

func (p *Provider) GetServerInfo() (*DockerInfo, error) {
	sysInfo, err := p.DockerClient.Info(p.Ctx)
	if err != nil {
		return nil, err
	}

	return &DockerInfo{
		CgroupDriver:    sysInfo.CgroupDriver,
		CgroupVersion:   sysInfo.CgroupVersion,
		StorageDriver:   sysInfo.Driver,
		MemoryLimit:     sysInfo.MemoryLimit,
		PidsLimit:       sysInfo.PidsLimit,
		CPUShares:       sysInfo.CPUShares,
		CPUNumber:       sysInfo.NCPU,
		SecurityOptions: sysInfo.SecurityOptions,
	}, nil
}

func NewClientProvider() (*Provider, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Provider{
		Ctx:          ctx,
		DockerClient: cli,
	}, nil
}
