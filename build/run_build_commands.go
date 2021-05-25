package build

import (
	"fmt"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
)

func (c *CloudBuilder) runBuildCommands() error {
	// send raw cluster file
	if err := c.SSH.Copy(c.RemoteHostIP, common.RawClusterfile, common.RawClusterfile); err != nil {
		return err
	}
	// apply k8s cluster
	apply := fmt.Sprintf("sealer apply -f %s", common.TmpClusterfile)
	err := c.SSH.CmdAsync(c.RemoteHostIP, apply)
	if err != nil {
		return fmt.Errorf("failed to run remote apply:%v", err)
	}
	// run local build command
	workdir := fmt.Sprintf(common.DefaultWorkDir, c.local.Cluster.Name)
	build := fmt.Sprintf(common.BuildClusterCmd, common.ExecBinaryFileName,
		c.local.KubeFileName, c.local.ImageName, common.LocalBuild)
	push := fmt.Sprintf(common.PushImageCmd, common.ExecBinaryFileName,
		c.local.ImageName)
	cmd := fmt.Sprintf("%s && %s", build, push)
	logger.Info("run remote shell %s", cmd)

	cmd = fmt.Sprintf("cd %s && %s", workdir, cmd)
	err = c.SSH.CmdAsync(c.RemoteHostIP, cmd)
	if err != nil {
		return fmt.Errorf("failed to run remote build and push:%v", err)
	}
	return nil
}
