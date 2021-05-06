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
	workdir := fmt.Sprintf(common.DefaultWorkDir, c.local.Cluster.Name)
	build := fmt.Sprintf(common.BuildClusterCmd, common.ExecBinaryFileName,
		c.local.KubeFileName, c.local.ImageName, common.LocalBuild)
	logger.Info("run remote build %s", build)

	cmd := fmt.Sprintf("cd %s && %s", workdir, build)
	err := c.SSH.CmdAsync(c.RemoteHostIP, cmd)
	if err != nil {
		return fmt.Errorf("failed to run remote build:%v", err)
	}
	return nil
}
