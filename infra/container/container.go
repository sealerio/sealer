package container

import (
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
)

func (c *DockerProvider) getUserNsMode() (container.UsernsMode, error) {
	sysInfo, err := c.DockerClient.Info(c.Ctx)
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

func (c *DockerProvider) setContainerMount(opts *CreateOptsForContainer) []mount.Mount {
	mounts := []mount.Mount{
		{
			Type:     mount.TypeBind,
			Source:   "/lib/modules",
			Target:   "/lib/modules",
			ReadOnly: true,
			BindOptions: &mount.BindOptions{
				Propagation: mount.PropagationRPrivate,
			},
		},
		{
			Type:     mount.TypeVolume,
			Source:   "",
			Target:   "/var",
			ReadOnly: false,
			VolumeOptions: &mount.VolumeOptions{
				DriverConfig: &mount.Driver{
					Name: "local",
				},
			},
		},
		{
			Type:     mount.TypeTmpfs,
			Source:   "",
			Target:   "/tmp",
			ReadOnly: false,
		},
		{
			Type:     mount.TypeTmpfs,
			Source:   "",
			Target:   "/run",
			ReadOnly: false,
		},
	}
	if opts.Mount != nil {
		mounts = append(mounts, *opts.Mount)
	}

	// only master0 need to bind root path
	if utils.IsFileExist(SealerImageRootPath) && opts.IsMaster0 {
		sealerMount := mount.Mount{
			Type:     mount.TypeBind,
			Source:   SealerImageRootPath,
			Target:   SealerImageRootPath,
			ReadOnly: false,
			BindOptions: &mount.BindOptions{
				Propagation: mount.PropagationRPrivate,
			},
		}
		mounts = append(mounts, sealerMount)
	}
	return mounts
}

func (c *DockerProvider) RunContainer(opts *CreateOptsForContainer) (string, error) {
	//docker run --hostname master1 --name master1
	//--privileged
	//--security-opt seccomp=unconfined --security-opt apparmor=unconfined
	//--tmpfs /tmp --tmpfs /run
	//--volume /var --volume /lib/modules:/lib/modules:ro
	//--device /dev/fuse
	//--detach --tty --restart=on-failure:1 --init=false sealer-io/sealer-base-image:latest
	// prepare run args according to docker server
	mod, _ := c.getUserNsMode()
	mounts := c.setContainerMount(opts)
	falseOpts := false
	resp, err := c.DockerClient.ContainerCreate(c.Ctx, &container.Config{
		Image:        c.ImageResource.DefaultName,
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
				c.NetworkResource.DefaultName: {
					NetworkID: c.NetworkResource.ID,
				},
			},
		}, nil, opts.ContainerName)

	if err != nil {
		return "", err
	}

	err = c.DockerClient.ContainerStart(c.Ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}
	logger.Info("create container %s successfully", opts.ContainerName)
	return resp.ID, nil
}

func (c *DockerProvider) GetContainerInfo(containerID string) (*Container, error) {
	resp, err := c.DockerClient.ContainerInspect(c.Ctx, containerID)
	if err != nil {
		return nil, err
	}
	return &Container{
		ContainerName:     resp.Name,
		ContainerIP:       resp.NetworkSettings.Networks[c.NetworkResource.DefaultName].IPAddress,
		ContainerHostName: resp.Config.Hostname,
		ContainerLabel:    resp.Config.Labels,
		Status:            resp.State.Status,
	}, nil
}

func (c *DockerProvider) GetContainerIDByIP(containerIP string) (string, error) {
	resp, err := c.DockerClient.ContainerList(c.Ctx, types.ContainerListOptions{})
	if err != nil {
		return "", err
	}

	for _, item := range resp {
		if containerIP == item.NetworkSettings.Networks[c.NetworkResource.DefaultName].IPAddress {
			return item.ID, nil
		}
	}
	return "", err
}

func (c *DockerProvider) RmContainer(containerID string) error {
	err := c.DockerClient.ContainerRemove(c.Ctx, containerID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})

	if err != nil {
		return err
	}

	logger.Info("delete container %s successfully", containerID)
	return nil
}

func (c *DockerProvider) RunSSHCMDInContainer(sshClient ssh.Interface, containerIP, cmd string) error {
	_, err := sshClient.Cmd(containerIP, cmd)
	return err
}
