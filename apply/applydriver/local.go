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

package applydriver

import (
	"fmt"

	"github.com/alibaba/sealer/utils/ssh"

	v1 "github.com/alibaba/sealer/types/api/v1"

	"github.com/alibaba/sealer/utils/platform"

	"github.com/alibaba/sealer/apply/processor"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/clusterfile"
	"github.com/alibaba/sealer/pkg/filesystem/cloudimage"
	"github.com/alibaba/sealer/pkg/image/store"
	"github.com/alibaba/sealer/pkg/runtime"
	v2 "github.com/alibaba/sealer/types/api/v2"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"

	"github.com/alibaba/sealer/pkg/client/k8s"
	"github.com/alibaba/sealer/pkg/image"
	"github.com/alibaba/sealer/utils"
)

// Applier cloud builder using cloud provider to build a cluster image
type Applier struct {
	ClusterDesired     *v2.Cluster
	ClusterCurrent     *v2.Cluster
	ClusterFile        clusterfile.Interface
	ImageManager       image.Service
	CloudImageMounter  cloudimage.Interface
	Client             *k8s.Client
	ImageStore         store.ImageStore
	CurrentClusterInfo *version.Info
}

func (c *Applier) Delete() (err error) {
	t := metav1.Now()
	c.ClusterDesired.DeletionTimestamp = &t
	return c.deleteCluster()
}

// Apply different actions between ClusterDesired and ClusterCurrent.
func (c *Applier) Apply() (err error) {
	// first time to init cluster
	if c.ClusterFile == nil {
		c.ClusterFile, err = clusterfile.NewClusterFile(c.ClusterDesired.GetAnnotationsByKey(common.ClusterfileName))
		if err != nil {
			return err
		}
	}
	if !utils.IsFileExist(common.DefaultKubeConfigFile()) {
		if err = c.initCluster(); err != nil {
			return err
		}
	} else {
		if err = c.reconcileCluster(); err != nil {
			return err
		}
	}

	return utils.SaveClusterInfoToFile(c.ClusterDesired, c.ClusterDesired.Name)
}

func (c *Applier) fillClusterCurrent() error {
	currentCluster, err := GetCurrentCluster(c.Client)
	if err != nil {
		return errors.Wrap(err, "get current cluster failed")
	}
	if currentCluster != nil {
		c.ClusterCurrent = c.ClusterDesired.DeepCopy()
		c.ClusterCurrent.Spec.Hosts = currentCluster.Spec.Hosts
	}
	return nil
}

func (c *Applier) mountClusterImage() error {
	imageName := c.ClusterDesired.Spec.Image
	//todo need to filter image by platform
	platsMap, err := ssh.GetClusterPlatform(c.ClusterDesired)
	if err != nil {
		return err
	}
	plats := []*v1.Platform{platform.GetDefaultPlatform()}
	for _, v := range platsMap {
		plat := v
		plats = append(plats, &plat)
	}
	err = c.ImageManager.PullIfNotExist(imageName, plats)
	if err != nil {
		return err
	}
	err = c.CloudImageMounter.MountImage(c.ClusterDesired)
	if err != nil {
		return err
	}
	return nil
}

func (c *Applier) unMountClusterImage() error {
	return c.CloudImageMounter.UnMountImage(c.ClusterDesired)
}

func (c *Applier) reconcileCluster() error {
	client, err := k8s.Newk8sClient()
	if err != nil {
		return err
	}
	c.Client = client
	info, err := c.Client.GetClusterVersion()
	if err != nil {
		return err
	}
	c.CurrentClusterInfo = info

	if err := c.fillClusterCurrent(); err != nil {
		return err
	}

	if err := c.mountClusterImage(); err != nil {
		return err
	}
	defer func() {
		if err := c.unMountClusterImage(); err != nil {
			logger.Warn("failed to umount image %s, %v", c.ClusterDesired.ClusterName, err)
		}
	}()

	//todo need to filter image by platform
	baseImage, err := c.ImageStore.GetByName(c.ClusterDesired.Spec.Image, platform.GetDefaultPlatform())
	if err != nil {
		return fmt.Errorf("failed to get base image err: %s", err)
	}
	// if no rootfs ,try to install applications.
	if baseImage.Spec.ImageConfig.ImageType == common.AppImage {
		return c.installApp()
	}

	mj, md := utils.GetDiffHosts(c.ClusterCurrent.GetMasterIPList(), c.ClusterDesired.GetMasterIPList())
	nj, nd := utils.GetDiffHosts(c.ClusterCurrent.GetNodeIPList(), c.ClusterDesired.GetNodeIPList())
	if len(mj) == 0 && len(md) == 0 && len(nj) == 0 && len(nd) == 0 {
		return c.upgrade()
	}
	return c.scaleCluster(mj, md, nj, nd)
}

func (c *Applier) scaleCluster(mj, md, nj, nd []string) error {
	logger.Info("Start to scale this cluster")
	logger.Debug("current cluster: master %s, worker %s", c.ClusterCurrent.GetMasterIPList(), c.ClusterCurrent.GetNodeIPList())

	scaleProcessor, err := processor.NewScaleProcessor(c.ClusterFile.GetKubeadmConfig(), c.ClusterFile, mj, md, nj, nd)
	if err != nil {
		return err
	}
	var cluster *v2.Cluster
	if !scaleProcessor.(*processor.ScaleProcessor).IsScaleUp {
		c, err := runtime.DecodeCRDFromFile(common.GetClusterWorkClusterfile(c.ClusterDesired.Name), common.Cluster)
		if err != nil {
			return err
		} else if c != nil {
			cluster = c.(*v2.Cluster)
		}
	} else {
		cluster = c.ClusterDesired
	}
	err = processor.NewExecutor(scaleProcessor).Execute(cluster)
	if err != nil {
		return err
	}

	logger.Info("Succeeded in scaling this cluster")

	return nil
}

func (c *Applier) Upgrade(upgradeImgName string) error {
	if err := c.initClusterfile(); err != nil {
		return err
	}
	if err := c.initK8sClient(); err != nil {
		return err
	}

	c.ClusterDesired.Spec.Image = upgradeImgName
	if err := c.mountClusterImage(); err != nil {
		return err
	}
	defer func() {
		if err := c.unMountClusterImage(); err != nil {
			logger.Warn("failed to umount image %s, %v", c.ClusterDesired.ClusterName, err)
		}
	}()
	return c.upgrade()
}

func (c *Applier) upgrade() error {
	runtimeInterface, err := runtime.NewDefaultRuntime(c.ClusterDesired, c.ClusterFile.GetKubeadmConfig())
	if err != nil {
		return fmt.Errorf("failed to init runtime, %v", err)
	}
	upgradeImgMeta, err := runtimeInterface.GetClusterMetadata()
	if err != nil {
		return fmt.Errorf("failed to get cluster metadata: %v", err)
	}

	if c.CurrentClusterInfo.GitVersion == upgradeImgMeta.Version {
		logger.Info("No upgrade required， image version and cluster version are both %s.", c.CurrentClusterInfo.GitVersion)
		return nil
	}
	logger.Info("Start to upgrade this cluster from version(%s) to version(%s)", c.CurrentClusterInfo.GitVersion, upgradeImgMeta.Version)

	upgradeProcessor, err := processor.NewUpgradeProcessor(platform.DefaultMountCloudImageDir(c.ClusterDesired.Name), runtimeInterface)
	if err != nil {
		return err
	}
	err = upgradeProcessor.Execute(c.ClusterDesired)
	if err != nil {
		return err
	}
	logger.Info("Succeeded in upgrading current cluster from version(%s) to version(%s)", c.CurrentClusterInfo.GitVersion, upgradeImgMeta.Version)
	return utils.SaveClusterInfoToFile(c.ClusterDesired, c.ClusterDesired.Name)
}

func (c *Applier) initClusterfile() (err error) {
	if c.ClusterFile != nil {
		return nil
	}
	c.ClusterFile, err = clusterfile.NewClusterFile(c.ClusterDesired.GetAnnotationsByKey(common.ClusterfileName))
	return err
}

func (c *Applier) initK8sClient() error {
	client, err := k8s.Newk8sClient()
	c.Client = client
	if err != nil {
		return err
	}
	info, err := client.GetClusterVersion()
	c.CurrentClusterInfo = info
	return err
}

func (c *Applier) installApp() error {
	rootfs := platform.DefaultMountCloudImageDir(c.ClusterDesired.Name)
	// use k8sClient to fetch current cluster version.
	info := c.CurrentClusterInfo

	clusterMetadata, err := runtime.LoadMetadata(rootfs)
	if err != nil {
		return err
	}
	if clusterMetadata != nil {
		if !VersionCompatible(info.GitVersion, clusterMetadata.KubeVersion) {
			return fmt.Errorf("incompatible application version, need: %s", clusterMetadata.KubeVersion)
		}
	}

	installProcessor, err := processor.NewInstallProcessor(c.ClusterFile)
	if err != nil {
		return err
	}
	err = processor.NewExecutor(installProcessor).Execute(c.ClusterDesired)
	if err != nil {
		return err
	}

	return nil
}

func (c *Applier) initCluster() error {
	logger.Info("Start to create a new cluster: master %s, worker %s", c.ClusterDesired.GetMasterIPList(), c.ClusterDesired.GetNodeIPList())
	createProcessor, err := processor.NewCreateProcessor(c.ClusterFile)
	if err != nil {
		return err
	}

	if err := processor.NewExecutor(createProcessor).Execute(c.ClusterDesired); err != nil {
		return err
	}

	logger.Info("Succeeded in creating a new cluster, enjoy it!")

	return nil
}

func (c *Applier) deleteCluster() error {
	deleteProcessor, err := processor.NewDeleteProcessor(c.ClusterFile)
	if err != nil {
		return err
	}
	if err := c.mountClusterImage(); err != nil {
		return err
	}
	//deleteProcessor to unmount image
	if err := processor.NewExecutor(deleteProcessor).Execute(c.ClusterDesired); err != nil {
		return err
	}

	logger.Info("Succeeded in deleting current cluster")

	return nil
}
