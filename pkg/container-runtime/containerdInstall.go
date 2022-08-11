package container_runtime

import (
	"net"
)

type ContainerdInstaller struct {
	DockerInstaller
}

func (c ContainerdInstaller) InstallOn(hosts []net.IP) (*Info, error) {

	return &c.info, nil
}

func (c ContainerdInstaller) UnInstallFrom(hosts []net.IP) error {

	return nil
}
