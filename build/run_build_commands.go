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

package build

import (
	"fmt"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
)

func (c *CloudBuilder) runBuildCommands() error {
	// send raw cluster file
	if !c.SSH.IsFileExist(c.RemoteHostIP, common.RawClusterfile) {
		if err := c.SSH.Copy(c.RemoteHostIP, common.RawClusterfile, common.RawClusterfile); err != nil {
			return err
		}
	}

	// apply k8s cluster
	apply := fmt.Sprintf("%s apply -f %s", common.RemoteSealerPath, common.TmpClusterfile)
	err := c.SSH.CmdAsync(c.RemoteHostIP, apply)
	if err != nil {
		return fmt.Errorf("failed to run remote apply:%v", err)
	}
	// run local build command
	workdir := fmt.Sprintf(common.DefaultWorkDir, c.local.Cluster.Name)
	build := fmt.Sprintf(common.BuildClusterCmd, common.RemoteSealerPath,
		c.local.KubeFileName, c.local.ImageName, common.LocalBuild, c.local.Context)
	push := fmt.Sprintf(common.PushImageCmd, common.RemoteSealerPath,
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
