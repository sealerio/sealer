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

package applytype

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"

	"github.com/alibaba/sealer/client/k8s"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/infra"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/runtime"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const ApplyCluster = "chmod +x %s && %s apply -f %s"

type CloudApplier struct {
	ClusterCurrent *v1.Cluster
	ClusterDesired *v1.Cluster
	Client         *k8s.Client
}

func (c *CloudApplier) ScaleDownNodes() (isScaleDown bool, err error) {
	logger.Info("desired master %s, current master %s, desired nodes %s, current nodes %s", c.ClusterDesired.Spec.Masters.Count,
		c.ClusterCurrent.Spec.Masters.Count,
		c.ClusterDesired.Spec.Nodes.Count,
		c.ClusterCurrent.Spec.Nodes.Count)
	if c.ClusterDesired.Spec.Masters.Count >= c.ClusterCurrent.Spec.Masters.Count &&
		c.ClusterDesired.Spec.Nodes.Count >= c.ClusterCurrent.Spec.Nodes.Count {
		return false, nil
	}

	mastersToJoin, mastersToDelete := utils.GetDiffHosts(c.ClusterCurrent.Spec.Masters, c.ClusterDesired.Spec.Masters)
	nodesToJoin, nodesToDelete := utils.GetDiffHosts(c.ClusterCurrent.Spec.Nodes, c.ClusterDesired.Spec.Nodes)
	if len(mastersToJoin) != 0 || len(nodesToJoin) != 0 {
		return false, fmt.Errorf("should not scale up and down at same time")
	}

	if err := DeleteNodes(c.Client, append(mastersToDelete, nodesToDelete...)); err != nil {
		return false, err
	}
	return true, nil
}

func (c *CloudApplier) Apply() error {
	// scale infra first.infra will update ClusterDesired filed.
	err := c.scaleInfra()
	if err != nil {
		return err
	}
	// first time to apply: create new cluster.
	if !utils.IsFileExist(common.DefaultKubeConfigFile()) {
		return c.runRemoteApply()
	}
	err = c.fillClusterCurrent()
	if err != nil {
		return err
	}
	//scale down
	scaleDown, err := c.ScaleDownNodes()
	if err != nil {
		return fmt.Errorf("failed to scale down nodes %v", err)
	}
	if scaleDown {
		// infra already delete the host, if continue to apply will not find the host and return ssh error
		logger.Info("scale the cluster success")
		return nil
	}
	// scale up
	err = c.runRemoteApply()
	if err != nil {
		return err
	}
	return nil
}

func (c *CloudApplier) Delete() error {
	t := metav1.Now()
	c.ClusterDesired.DeletionTimestamp = &t
	host := c.ClusterDesired.GetAnnotationsByKey(common.Eip)
	err := c.scaleInfra()
	if err != nil {
		return err
	}
	if err = utils.RemoveFileContent(common.EtcHosts, fmt.Sprintf("%s %s", host, common.APIServerDomain)); err != nil {
		logger.Warn(err)
	}

	if err = utils.CleanFiles(common.DefaultKubeConfigDir(), common.GetClusterWorkDir(c.ClusterDesired.Name), common.TmpClusterfile, common.KubectlPath); err != nil {
		logger.Warn(err)
		return nil
	}

	return nil
}

func (c *CloudApplier) scaleInfra() error {
	logger.Info("start to scale the cluster infra")
	cloudProvider, err := infra.NewDefaultProvider(c.ClusterDesired)
	if err != nil {
		return err
	}
	if cloudProvider == nil {
		return fmt.Errorf("new cloud provider failed")
	}
	err = cloudProvider.Apply()
	if err != nil {
		return err
	}
	return utils.SaveClusterfile(c.ClusterDesired)
}

func (c *CloudApplier) fillClusterCurrent() error {
	client, err := k8s.Newk8sClient()
	if err != nil {
		return err
	}
	c.Client = client
	currentCluster, err := GetCurrentCluster(client)
	if err != nil {
		return errors.Wrap(err, "get current cluster failed")
	}

	if currentCluster != nil {
		c.ClusterCurrent = c.ClusterDesired.DeepCopy()
		c.ClusterCurrent.Spec.Masters = currentCluster.Spec.Masters
		c.ClusterCurrent.Spec.Nodes = currentCluster.Spec.Nodes
	}
	return nil
}

func (c *CloudApplier) runRemoteApply() error {
	client, err := ssh.NewSSHClientWithCluster(c.ClusterDesired)
	if err != nil {
		return fmt.Errorf("prepare cluster ssh client failed %v", err)
	}

	err = generateTmpClusterfile(c.ClusterDesired)
	if err != nil {
		return fmt.Errorf("failed to generate TmpClusterfile, %v", err)
	}
	defer func() {
		if err := utils.CleanFiles(common.TmpClusterfile); err != nil {
			logger.Error("failed to clean %s, err: %v", common.TmpClusterfile, err)
		}
	}()

	err = runtime.PreInitMaster0(client.SSH, client.Host)
	if err != nil {
		return err
	}

	err = client.SSH.CmdAsync(client.Host, fmt.Sprintf(ApplyCluster, common.RemoteSealerPath, common.RemoteSealerPath, common.TmpClusterfile))
	if err != nil {
		return err
	}

	err = runtime.GetKubectlAndKubeconfig(client.SSH, client.Host)
	if err != nil {
		return fmt.Errorf("failed to copy kubeconfig and kubectl %v", err)
	}
	return nil
}

func generateTmpClusterfile(cluster *v1.Cluster) error {
	cluster.Spec.Provider = common.BAREMETAL
	clusterfile := cluster.GetAnnotationsByKey(common.ClusterfileName)
	data, err := yaml.Marshal(cluster)
	if err != nil {
		return fmt.Errorf("failed to marshal cluster, %v", err)
	}
	if clusterfile == "" {
		return utils.WriteFile(common.TmpClusterfile, data)
	}
	appendData := [][]byte{data}
	plugins, err := utils.DecodePlugins(clusterfile)
	if err != nil {
		return err
	}
	for _, plugin := range plugins {
		data, err := yaml.Marshal(plugin)
		if err != nil {
			return fmt.Errorf("failed to marshal plugin, %v", err)
		}
		appendData = append(appendData, []byte("---\n"), data)
	}

	configs, err := utils.DecodeConfigs(clusterfile)
	if err != nil {
		return err
	}
	for _, config := range configs {
		data, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal config, %v", err)
		}
		appendData = append(appendData, []byte("---\n"), data)
	}

	err = utils.WriteFile(common.TmpClusterfile, bytes.Join(appendData, []byte("")))
	if err != nil {
		return err
	}
	return nil
}
