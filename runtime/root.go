/*
Copyright 2021 cuisongliu@qq.com.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package runtime

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

func (d *Default) CopyFilesOnRoot(cluster *v1.Cluster) error {
	d.addMaster0Host(cluster)
	d.copyKubeConfig(cluster)
	d.copyKubectl(cluster)
	return nil
}

func (d *Default) addMaster0Host(cluster *v1.Cluster) {
	content, err := utils.ReadAll(common.EtcHosts)
	if err != nil {
		return
	}
	if !strings.Contains(string(content), common.APIServerDomain) {
		err = utils.AppendFile(common.EtcHosts, fmt.Sprintf("%s %s", cluster.Spec.Masters.IPList[0], common.APIServerDomain))
		if err != nil {
			logger.Warn("append master0 host to etc hosts failed: %v", err)
		}
	}
}
func (d *Default) copyKubeConfig(cluster *v1.Cluster) {
	if err := utils.MkDirIfNotExists(common.DefaultKubeConfigDir()); err != nil {
		logger.Warn("mkdir kube dir failed: %v", err)
	}
	adminConf := filepath.Join(common.DefaultClusterRootfsDir, cluster.Name, "admin.conf")
	if !utils.IsFileExist(adminConf) {
		adminConf = common.KubeAdminConf
	}
	kubeConfig := filepath.Join(common.DefaultKubeConfigDir(), "config")
	if !utils.IsFileExist(kubeConfig) {
		_, err := utils.CopySingleFile(adminConf, kubeConfig)
		if err != nil {
			logger.Warn("copy kube config failed: %v", err)
		}
	}
}
func (d *Default) copyKubectl(cluster *v1.Cluster) {
	if !utils.IsFileExist(common.KubectlPath) {
		clusterTmpRootfsDir := filepath.Join("/tmp", cluster.Name)
		kubectl := filepath.Join(clusterTmpRootfsDir, "bin", "kubectl")
		if utils.IsFileExist(kubectl) {
			_, err := utils.CopySingleFile(kubectl, common.KubectlPath)
			if err != nil {
				logger.Warn("copy kubectl failed: %v", err)
			}
			err = utils.Cmd("chmod", "+x", common.KubectlPath)
			if err != nil {
				logger.Warn("chmod a+x kubectl failed: %v", err)
			}
		}
	}
}
func (d *Default) CleanFilesOnRoot(cluster *v1.Cluster) error {
	d.deleteClusterConfigOnRoot(cluster)
	d.deleteMaster0HostOnRoot(cluster)
	return nil
}

func (d *Default) deleteClusterConfigOnRoot(cluster *v1.Cluster) {
	clusterTmpRootfsDir := filepath.Join("/tmp", cluster.Name)
	if err := utils.CleanFiles(common.DefaultKubeConfigDir(), clusterTmpRootfsDir, common.GetClusterRootfsDir(cluster.Name), common.GetClusterWorkDir(cluster.Name), common.TmpClusterfile, common.KubectlPath); err != nil {
		logger.Warn(err)
	}
}
func (d *Default) deleteMaster0HostOnRoot(cluster *v1.Cluster) {
	err := utils.RemoveFileContent(common.EtcHosts, fmt.Sprintf("%s %s", cluster.Spec.Masters.IPList[0], common.APIServerDomain))
	if err != nil {
		logger.Info("remove /etc/host failed: %v", err)
	}
}
