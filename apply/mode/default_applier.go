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

package mode

import (
	"path/filepath"

	"github.com/alibaba/sealer/apply/base"
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
	Client         *k8s.Client
}

func (c *Applier) Delete() (err error) {
	t := metav1.Now()
	c.ClusterDesired.DeletionTimestamp = &t
	return c.Apply()
}

// Apply different actions between ClusterDesired and ClusterCurrent.
func (c *Applier) Apply() (err error) {
	//delete cluster
	if c.ClusterDesired.DeletionTimestamp != nil {
		return c.deleteCluster()
	}

	err = c.fillClusterCurrent()
	if err != nil {
		return err
	}

	// first time to init cluster
	if c.ClusterCurrent == nil {
		return c.initCluster()
	}
	// change same name of the cluster, such as upgrade,scale,install app on an existed cluster.
	// k8s version change: upgradeCluster
	// ip change: scaleCluster
	// no k8s version: install app
	currentMetadata, err := runtime.LoadMetadata(filepath.Join(common.DefaultTheClusterRootfsDir(c.ClusterDesired.Name),
		common.DefaultMetadataName))
	if err != nil {
		return err
	}

	imageName := c.ClusterDesired.Spec.Image
	err = c.ImageManager.PullIfNotExist(imageName)
	if err != nil {
		return err
	}
	err = c.FileSystem.MountImage(c.ClusterDesired)
	if err != nil {
		return err
	}
	defer func() {
		err = c.FileSystem.UnMountImage(c.ClusterDesired)
		if err != nil {
			logger.Warn("failed to umount image %s", c.ClusterDesired.ClusterName)
		}
	}()

	desiredMetadata, err := runtime.LoadMetadata(filepath.Join(common.DefaultMountCloudImageDir(c.ClusterDesired.Name),
		common.DefaultMetadataName))
	if err != nil {
		return err
	}
	//if desiredMetadata.Version==""{
	//	//install app
	//  c.upgradeCluster(c.ClusterDesired)
	//}
	if currentMetadata.Version != desiredMetadata.Version {
		logger.Info("different metadata (old %s,new %s) version will upgrade current cluster",
			currentMetadata.Version, desiredMetadata.Version)
		if err = c.upgradeCluster(); err != nil {
			return err
		}
	}
	if err = c.scaleCluster(); err != nil {
		return err
	}
	return nil
}

func (c *Applier) fillClusterCurrent() error {
	currentCluster, err := GetCurrentCluster(c.Client)
	if err != nil {
		return errors.Wrap(err, "get current cluster failed")
	}
	if currentCluster != nil {
		c.ClusterCurrent = c.ClusterDesired.DeepCopy()
		c.ClusterCurrent.Spec.Masters = currentCluster.Spec.Masters
		c.ClusterCurrent.Spec.Nodes = currentCluster.Spec.Nodes
	}
	err = utils.SaveClusterfile(c.ClusterDesired)
	if err != nil {
		return err
	}
	return nil
}

func (c *Applier) upgradeCluster() error {
	applier, err := base.NewUpgradeApply(c.FileSystem)
	if err != nil {
		return err
	}
	return applier.DoApply(c.ClusterDesired)
}

func (c *Applier) scaleCluster() error {
	mj, md := utils.GetDiffHosts(c.ClusterCurrent.Spec.Masters, c.ClusterDesired.Spec.Masters)
	nj, nd := utils.GetDiffHosts(c.ClusterCurrent.Spec.Nodes, c.ClusterDesired.Spec.Nodes)
	applier, err := base.NewScaleApply(c.FileSystem, md, mj, nd, nj)
	if err != nil {
		return err
	}
	return applier.DoApply(c.ClusterDesired)
}

func (c *Applier) initCluster() error {
	logger.Info("Current cluster is nil will create new cluster")
	applier, err := base.NewInitApply()
	if err != nil {
		return err
	}
	return applier.DoApply(c.ClusterDesired)
}

func (c *Applier) deleteCluster() error {
	logger.Info("Current cluster is nil will delete cluster")
	applier, err := base.NewDeleteApply()
	if err != nil {
		return err
	}
	return applier.DoApply(c.ClusterDesired)
}
