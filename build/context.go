package build

import (
	"fmt"
	"gitlab.alibaba-inc.com/seadent/pkg/common"
	"gitlab.alibaba-inc.com/seadent/pkg/logger"
	"gitlab.alibaba-inc.com/seadent/pkg/runtime"
	"gitlab.alibaba-inc.com/seadent/pkg/utils"
)

//sendBuildContext:send local build context to remote server
func (c *CloudBuilder) sendBuildContext() (err error) {
	// if remote cluster already exist,no need to pre init master0
	if !c.SSH.IsFileExist(c.RemoteHostIp, common.RemoteSealerPath) {
		err = runtime.PreInitMaster0(c.SSH, c.RemoteHostIp)
		if err != nil {
			return fmt.Errorf("failed to prepare cluster env %v", err)
		}
	}

	// tar local build context
	tarFileName := fmt.Sprintf(common.TmpTarFile, c.local.Image.Spec.ID)
	if _, isExist := utils.CheckCmdIsExist("tar"); !isExist {
		return fmt.Errorf("local server muster support tar cmd")
	}
	if _, err := utils.RunSimpleCmd(fmt.Sprintf(common.ZipCmd, tarFileName, c.local.Context)); err != nil {
		return fmt.Errorf("failed to create context file: %v", err)
	}
	// send to remote server
	workdir := fmt.Sprintf(common.DefaultWorkDir, c.local.Cluster.Name)
	if err := c.SSH.Copy(c.RemoteHostIp, tarFileName, tarFileName); err != nil {
		return err
	}
	// unzip remote context
	err = c.SSH.CmdAsync(c.RemoteHostIp, fmt.Sprintf(common.UnzipCmd, workdir, tarFileName, workdir))
	if err != nil {
		return err
	}
	logger.Info("send build context to %s success !", c.RemoteHostIp)
	return nil
}
