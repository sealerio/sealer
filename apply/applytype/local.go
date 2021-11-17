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
	"path/filepath"

	"github.com/alibaba/sealer/apply/applyentity"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/runtime"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/sealer/client/k8s"
	"github.com/alibaba/sealer/filesystem"
	"github.com/alibaba/sealer/image"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

// Applier cloud builder using cloud provider to build a cluster image
type Applier struct {
	ClusterDesired *v1.Cluster
	ClusterCurrent *v1.Cluster
	ImageManager   image.Service
	FileSystem     filesystem.Interface
}

func (c *Applier) Delete() (err error) {
	t := metav1.Now()
	c.ClusterDesired.DeletionTimestamp = &t
	return c.deleteCluster()
}

// Apply different actions between ClusterDesired and ClusterCurrent.
func (c *Applier) Apply() (err error) {
	err = utils.SaveClusterfile(c.ClusterDesired)
	if err != nil {
		return err
	}
	// first time to init cluster
	if !utils.IsFileExist(common.DefaultKubeConfigFile()) {
		return c.initCluster()
	}
	return c.changeCluster()
}

func (c *Applier) fillClusterCurrent() error {
	client, err := k8s.Newk8sClient()
	if err != nil {
		return err
	}
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

func (c *Applier) mountClusterImage() error {
	imageName := c.ClusterDesired.Spec.Image
	err := c.ImageManager.PullIfNotExist(imageName)
	if err != nil {
		return err
	}
	err = c.FileSystem.MountImage(c.ClusterDesired)
	if err != nil {
		return err
	}
	return nil
}

func (c *Applier) unMountClusterImage() error {
	return c.FileSystem.UnMountImage(c.ClusterDesired)
}

func (c *Applier) changeCluster() error {
	err := c.fillClusterCurrent()
	if err != nil {
		return err
	}
	err = c.mountClusterImage()
	if err != nil {
		return err
	}
	defer func() {
		err = c.unMountClusterImage()
		if err != nil {
			logger.Warn("failed to umount image %s", c.ClusterDesired.ClusterName)
		}
	}()

	mj, md := utils.GetDiffHosts(c.ClusterCurrent.Spec.Masters.IPList, c.ClusterDesired.Spec.Masters.IPList)
	nj, nd := utils.GetDiffHosts(c.ClusterCurrent.Spec.Nodes.IPList, c.ClusterDesired.Spec.Nodes.IPList)

	if err = c.scaleCluster(mj, md, nj, nd); err != nil {
		return err
	}
	if err = c.upgradeCluster(mj, nj); err != nil {
		return err
	}
	return nil
}

func (c *Applier) scaleCluster(mj, md, nj, nd []string) error {
	if len(mj) == 0 && len(md) == 0 && len(nj) == 0 && len(nd) == 0 {
		return nil
	}
	applier, err := applyentity.NewScaleApply(c.FileSystem, mj, md, nj, nd)
	if err != nil {
		return err
	}

	err = applier.DoApply(c.ClusterDesired)
	if err != nil {
		return err
	}
	return nil
}

func (c *Applier) upgradeCluster(mj, nj []string) error {
	currentMetadata, err := runtime.LoadMetadata(filepath.Join(common.DefaultTheClusterRootfsDir(c.ClusterDesired.Name),
		common.DefaultMetadataName))
	if err != nil {
		return err
	}

	desiredMetadata, err := runtime.LoadMetadata(filepath.Join(common.DefaultMountCloudImageDir(c.ClusterDesired.Name),
		common.DefaultMetadataName))
	if err != nil {
		return err
	}

	if currentMetadata.Version == desiredMetadata.Version {
		return nil
	}

	logger.Info("different metadata (old %s,new %s) version will upgrade current cluster",
		currentMetadata.Version, desiredMetadata.Version)
	//if currentMetadata.Version==""{
	//	//install app
	//}

	applier, err := applyentity.NewUpgradeApply(c.FileSystem, mj, nj)
	if err != nil {
		return err
	}
	err = applier.DoApply(c.ClusterDesired)
	if err != nil {
		return err
	}

	return nil
}

func (c *Applier) initCluster() error {
	logger.Info("Current cluster is nil will create new cluster")
	applier, err := applyentity.NewInitApply()
	if err != nil {
		return err
	}
	return applier.DoApply(c.ClusterDesired)
}

func (c *Applier) deleteCluster() error {
	logger.Info("Current cluster is nil will delete cluster")
	applier, err := applyentity.NewDeleteApply()
	if err != nil {
		return err
	}
	return applier.DoApply(c.ClusterDesired)
}
