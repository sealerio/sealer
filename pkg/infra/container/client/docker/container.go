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
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/pkg/infra/container/client"
)

const (
	CgroupNoneDriver = "none"
)

func (p *Provider) getUserNsMode() (container.UsernsMode, error) {
	sysInfo, err := p.DockerClient.Info(p.Ctx)
	if err != nil {
		return "", err
	}

	var usernsMode container.UsernsMode
	for _, opt := range sysInfo.SecurityOptions {
		if opt == "name=userns" {
			usernsMode = "host"
		}
	}
	return usernsMode, err
}

func (p *Provider) setContainerMount(opts *client.CreateOptsForContainer) []mount.Mount {
	mounts := DefaultMounts()
	if opts.Mount != nil {
		mounts = append(mounts, opts.Mount...)
	}
	return mounts
}

func (p *Provider) RunContainer(opts *client.CreateOptsForContainer) (string, error) {
	//docker run --hostname master1 --name master1
	//--privileged
	//--security-opt seccomp=unconfined --security-opt apparmor=unconfined
	//--tmpfs /tmp --tmpfs /run
	//--volume /var --volume /lib/modules:/lib/modules:ro
	//--device /dev/fuse
	//--detach --tty --restart=on-failure:1 --init=false sealer-io/sealer-base-image:latest
	networkID, err := p.PrepareNetworkResource(opts.NetworkName)
	if err != nil {
		return "", err
	}

	_, err = p.PullImage(opts.ImageName)
	if err != nil {
		return "", err
	}

	mod, _ := p.getUserNsMode()
	mounts := p.setContainerMount(opts)
	falseOpts := false
	resp, err := p.DockerClient.ContainerCreate(p.Ctx, &container.Config{
		Image:        opts.ImageName,
		Tty:          true,
		Labels:       opts.ContainerLabel,
		Hostname:     opts.ContainerHostName,
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
	},
		&container.HostConfig{
			UsernsMode: mod,
			SecurityOpt: []string{
				"seccomp=unconfined", "apparmor=unconfined",
			},
			RestartPolicy: container.RestartPolicy{
				Name:              "on-failure",
				MaximumRetryCount: 1,
			},
			Init:         &falseOpts,
			CgroupnsMode: "host",
			Privileged:   true,
			Mounts:       mounts,
		}, &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				opts.NetworkName: {
					NetworkID: networkID,
				},
			},
		}, nil, opts.ContainerName)

	if err != nil {
		return "", err
	}

	err = p.DockerClient.ContainerStart(p.Ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}
	logrus.Infof("create container %s successfully", opts.ContainerName)
	return resp.ID, nil
}

func (p *Provider) GetContainerInfo(containerID string, networkName string) (*client.Container, error) {
	resp, err := p.DockerClient.ContainerInspect(p.Ctx, containerID)
	if err != nil {
		return nil, err
	}
	return &client.Container{
		ContainerName:     resp.Name,
		ContainerIP:       resp.NetworkSettings.Networks[networkName].IPAddress,
		ContainerHostName: resp.Config.Hostname,
		ContainerLabel:    resp.Config.Labels,
		Status:            resp.State.Status,
	}, nil
}

func (p *Provider) GetContainerIDByIP(containerIP string, networkName string) (string, error) {
	resp, err := p.DockerClient.ContainerList(p.Ctx, types.ContainerListOptions{})
	if err != nil {
		return "", err
	}

	for _, item := range resp {
		if net, ok := item.NetworkSettings.Networks[networkName]; ok {
			if containerIP == net.IPAddress {
				return item.ID, nil
			}
		}
	}
	return "", err
}

func (p *Provider) RmContainer(containerID string) error {
	err := p.DockerClient.ContainerRemove(p.Ctx, containerID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})

	if err != nil {
		return err
	}

	logrus.Infof("delete container %s successfully", containerID)
	return nil
}

func (p *Provider) GetServerInfo() (*client.DockerInfo, error) {
	sysInfo, err := p.DockerClient.Info(p.Ctx)
	if err != nil {
		return nil, err
	}

	var dInfo client.DockerInfo

	// When CgroupDriver == "none", the MemoryLimit/PidsLimit/CPUShares
	// values are meaningless and need to be considered false.
	// https://github.com/moby/moby/issues/42151
	dInfo.CgroupVersion = sysInfo.CgroupVersion
	dInfo.StorageDriver = sysInfo.Driver
	dInfo.SecurityOptions = sysInfo.SecurityOptions
	dInfo.CgroupDriver = sysInfo.CgroupDriver
	if sysInfo.CgroupDriver == CgroupNoneDriver {
		return &dInfo, nil
	}
	dInfo.MemoryLimit = sysInfo.MemoryLimit
	dInfo.PidsLimit = sysInfo.PidsLimit
	dInfo.CPUShares = sysInfo.CPUShares
	dInfo.CPUNumber = sysInfo.NCPU
	return &dInfo, nil
}
