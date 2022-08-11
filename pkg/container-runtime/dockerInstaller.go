package container_runtime

import (
	"fmt"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/registry"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/platform"
	"github.com/sealerio/sealer/utils/ssh"
	"net"
)

type DockerInstaller struct {
	info    Info
	cluster *v2.Cluster
}

func (d DockerInstaller) InstallOn(hosts []net.IP) (*Info, error) {
	rootfs := fmt.Sprintf(common.DefaultTheClusterRootfsDir(d.cluster.Name))
	for ip := range hosts {
		IP := net.ParseIP(string(ip))
		ssh, err := ssh.NewStdoutSSHClient(IP, d.cluster)
		if err != nil {
			return nil, err
		}
		registryConfig := registry.GetConfig(platform.DefaultMountClusterImageDir(d.cluster.Name), IP)
		initCmd := fmt.Sprintf(RemoteChmod, rootfs, registryConfig.Domain, registryConfig.Port, d.info.conf.CgroupDriver, d.info.conf.LimitNofile)
		err = ssh.CmdAsync(IP, initCmd)
		if err != nil {
			return nil, err
		}
	}
	return &d.info, nil
}

func (d DockerInstaller) UnInstallFrom(hosts []net.IP) error {
	for ip := range hosts {
		IP := net.ParseIP(string(ip))
		client, err := ssh.NewStdoutSSHClient(IP, d.cluster)
		if err != nil {
			return fmt.Errorf("new ssh client failed: %s", err)
		}
		err = client.CmdAsync(IP, CleanCmd)
		if err != nil {
			return fmt.Errorf("remote exec clean cmd failed: %s", err)
		}
	}
	return nil
}
