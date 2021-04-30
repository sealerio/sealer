package runtime

import (
	"fmt"
	"strings"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
)

// if v1 >= v2 return true, else return false
func VersionCompare(v1, v2 string) bool {
	v1 = strings.Replace(v1, "v", "", -1)
	v2 = strings.Replace(v2, "v", "", -1)
	v1 = strings.Split(v1, "-")[0]
	v2 = strings.Split(v2, "-")[0]
	v1List := strings.Split(v1, ".")
	v2List := strings.Split(v2, ".")

	if len(v1List) != 3 || len(v2List) != 3 {
		logger.Error("error version format %s %s", v1, v2)
		return false
	}
	if v1List[0] > v2List[0] {
		return true
	} else if v1List[0] < v2List[0] {
		return false
	}
	if v1List[1] > v2List[1] {
		return true
	} else if v1List[1] < v2List[1] {
		return false
	}
	if v1List[2] > v2List[2] {
		return true
	}
	return true
}

func PreInitMaster0(sshClient ssh.Interface, remoteHostIP string) error {
	err := ssh.WaitSSHReady(sshClient, remoteHostIP)
	if err != nil {
		return fmt.Errorf("apply cloud cluster failed: %s", err)
	}
	// send sealer and cluster file to remote host
	sealerPath := utils.ExecutableFilePath()
	err = sshClient.Copy(remoteHostIP, sealerPath, common.RemoteSealerPath)
	if err != nil {
		return fmt.Errorf("send seautil to remote host %s failed:%v", remoteHostIP, err)
	}
	err = sshClient.CmdAsync(remoteHostIP, fmt.Sprintf(common.ChmodCmd, common.RemoteSealerPath))
	if err != nil {
		return fmt.Errorf("chmod +x seautil on remote host %s failed:%v", remoteHostIP, err)
	}
	logger.Info("send sealer cmd to %s success !", remoteHostIP)

	// send tmp cluster file
	err = sshClient.Copy(remoteHostIP, common.TmpClusterfile, common.TmpClusterfile)
	if err != nil {
		return fmt.Errorf("send cluster file to remote host %s failed:%v", remoteHostIP, err)
	}
	logger.Info("send cluster file to %s success !", remoteHostIP)

	// send register login info
	err = sshClient.Copy(remoteHostIP, common.DefaultRegistryAuthConfigDir(), common.DefaultRegistryAuthConfigDir())
	if err != nil {
		return fmt.Errorf("send register config to remote host %s failed:%v", remoteHostIP, err)
	}
	logger.Info("send register info to %s success !", remoteHostIP)

	return nil
}
