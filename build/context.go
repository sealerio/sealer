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
	"os"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/runtime"
	"github.com/alibaba/sealer/utils"
)

//sendBuildContext:send local build context to remote server
func (c *CloudBuilder) sendBuildContext() (err error) {
	// if remote cluster already exist,no need to pre init master0
	if !c.SSH.IsFileExist(c.RemoteHostIP, common.RemoteSealerPath) {
		err = runtime.PreInitMaster0(c.SSH, c.RemoteHostIP)
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
	defer func() {
		if err = os.Remove(tarFileName); err != nil {
			logger.Warn("failed to cleanup local temp file %s:%v", tarFileName, err)
		}
	}()
	// send to remote server
	workdir := fmt.Sprintf(common.DefaultWorkDir, c.local.Cluster.Name)
	if err := c.SSH.Copy(c.RemoteHostIP, tarFileName, tarFileName); err != nil {
		return err
	}
	// unzip remote context
	err = c.SSH.CmdAsync(c.RemoteHostIP, fmt.Sprintf(common.UnzipCmd, workdir, tarFileName, workdir))
	if err != nil {
		return err
	}
	logger.Info("send build context to %s success !", c.RemoteHostIP)
	return nil
}
