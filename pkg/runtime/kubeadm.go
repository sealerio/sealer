package runtime

import (
	"github.com/alibaba/sealer/logger"
	"strings"
)

var (
	ContainerdShell = `if grep "SystemdCgroup = true"  /etc/containerd/config.toml &> /dev/null; then  
driver=systemd
else
driver=cgroupfs
fi
echo ${driver}`
	DockerShell = `driver=$(docker info -f "{{.CgroupDriver}}")
	echo "${driver}"`
)

func (k *KubeadmRuntime) setKubeadmAPIAndCriSocketByVersion() {
	switch {
	case VersionCompare(k.Metadata.Version, V1150) && !VersionCompare(k.Metadata.Version, V1200):
		k.InitConfiguration.NodeRegistration.CRISocket = DefaultDockerCRISocket
		k.InitConfiguration.APIVersion = KubeadmV1beta2
		k.ClusterConfiguration.APIVersion = KubeadmV1beta2
	// kubernetes gt 1.20, use Containerd instead of docker
	case VersionCompare(k.Metadata.Version, V1200):
		k.InitConfiguration.NodeRegistration.CRISocket = DefaultContainerdCRISocket
		k.InitConfiguration.APIVersion = KubeadmV1beta2
		k.ClusterConfiguration.APIVersion = KubeadmV1beta2
	default:
		// Compatible with versions 1.14 and 1.13. but do not recommended.
		k.InitConfiguration.NodeRegistration.CRISocket = DefaultDockerCRISocket
		k.InitConfiguration.APIVersion = KubeadmV1beta1
		k.ClusterConfiguration.APIVersion = KubeadmV1beta1
	}
}

// getCgroupDriverFromShell is get nodes container runtime cgroup by shell.
func (k *KubeadmRuntime) getCgroupDriverFromShell(node string) string {
	var cmd string
	if VersionCompare(k.Metadata.Version, V1200) {
		cmd = ContainerdShell
	} else {
		cmd = DockerShell
	}
	driver := k.CmdToString(node, cmd, " ")
	//driver, err := k.SSH.CmdToString(node, cmd, " ")
	if driver == "" {
		// by default if we get wrong output we set it default systemd?
		logger.Error("failed to get nodes [%s] cgroup driver", node)
		driver = DefaultSystemdCgroupDriver
	}
	driver = strings.TrimSpace(driver)
	logger.Debug("get nodes [%s] cgroup driver is [%s]", node, driver)
	return driver
}
