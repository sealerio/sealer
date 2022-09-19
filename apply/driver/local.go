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

package driver

import (
	"fmt"
	"net"

	"github.com/sealerio/sealer/apply/processor"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/client/k8s"
	"github.com/sealerio/sealer/pkg/clusterfile"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/filesystem/clusterimage"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/runtime"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils"
	osi "github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/platform"
	"github.com/sealerio/sealer/utils/ssh"
	"github.com/sealerio/sealer/utils/strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
)

// Applier cloud builder using cloud provider to build a ClusterImage
type Applier struct {
	ClusterDesired      *v2.Cluster
	ClusterCurrent      *v2.Cluster
	ClusterFile         clusterfile.Interface
	ImageEngine         imageengine.Interface
	ClusterImageMounter clusterimage.Interface
	Client              *k8s.Client
	CurrentClusterInfo  *version.Info
}

func (applier *Applier) Delete() (err error) {
	t := metav1.Now()
	applier.ClusterDesired.DeletionTimestamp = &t
	return applier.deleteCluster()
}

// Apply different actions between ClusterDesired and ClusterCurrent.
func (applier *Applier) Apply() (err error) {
	// first time to init cluster
	if applier.ClusterFile == nil {
		applier.ClusterFile, err = clusterfile.NewClusterFile(applier.ClusterDesired.GetAnnotationsByKey(common.ClusterfileName))
		if err != nil {
			return err
		}
	}
	if !osi.IsFileExist(common.DefaultKubeConfigFile()) {
		if err = applier.initCluster(); err != nil {
			return err
		}
	} else {
		if err = applier.reconcileCluster(); err != nil {
			return err
		}
	}

	return clusterfile.SaveToDisk(applier.ClusterDesired, applier.ClusterDesired.Name)
}

func (applier *Applier) fillClusterCurrent() error {
	currentCluster, err := GetCurrentCluster(applier.Client)
	if err != nil {
		return errors.Wrap(err, "get current cluster failed")
	}
	if currentCluster != nil {
		applier.ClusterCurrent = applier.ClusterDesired.DeepCopy()
		applier.ClusterCurrent.Spec.Hosts = currentCluster.Spec.Hosts
	}
	return nil
}

func (applier *Applier) mountClusterImage() error {
	imageName := applier.ClusterDesired.Spec.Image
	platsMap, err := ssh.GetClusterPlatform(applier.ClusterDesired)
	if err != nil {
		return err
	}

	platVisit := map[string]struct{}{}
	platVisit[platform.GetDefaultPlatform().ToString()] = struct{}{}
	for _, v := range platsMap {
		platVisit[v.ToString()] = struct{}{}
	}

	plats := []string{}
	for k := range platVisit {
		plats = append(plats, k)
	}
	// TODO optimize image engine caller.
	for _, plat := range plats {
		err = applier.ImageEngine.Pull(&imagecommon.PullOptions{
			Quiet:      false,
			TLSVerify:  true,
			PullPolicy: "missing",
			Image:      imageName,
			Platform:   plat,
		})

		if err != nil {
			return err
		}
	}

	err = applier.ClusterImageMounter.MountImage(applier.ClusterDesired)
	if err != nil {
		return err
	}
	return nil
}

func (applier *Applier) unMountClusterImage() error {
	return applier.ClusterImageMounter.UnMountImage(applier.ClusterDesired)
}

func (applier *Applier) reconcileCluster() error {
	client, err := k8s.Newk8sClient()
	if err != nil {
		return err
	}
	applier.Client = client
	info, err := applier.Client.GetClusterVersion()
	if err != nil {
		return err
	}
	applier.CurrentClusterInfo = info

	if err := applier.fillClusterCurrent(); err != nil {
		return err
	}

	if err := applier.mountClusterImage(); err != nil {
		return err
	}
	defer func() {
		if err := applier.unMountClusterImage(); err != nil {
			logrus.Warnf("failed to umount cluster(%s): %v", applier.ClusterDesired.Name, err)
		}

		if err := applier.ImageEngine.RemoveContainer(&imagecommon.RemoveContainerOptions{
			All: true,
		}); err != nil {
			logrus.Warnf("failed to clean the buildah containers: %v", err)
		}
	}()
	image := applier.ClusterDesired.Spec.Image
	// TODO
	imageExtension, err := applier.ImageEngine.GetSealerImageExtension(&imagecommon.GetImageAnnoOptions{ImageNameOrID: image})
	if err != nil {
		return err
	}

	if imageExtension.ImageType == common.AppImage {
		return applier.installApp()
	}

	mj, md := strings.Diff(applier.ClusterCurrent.GetMasterIPList(), applier.ClusterDesired.GetMasterIPList())
	nj, nd := strings.Diff(applier.ClusterCurrent.GetNodeIPList(), applier.ClusterDesired.GetNodeIPList())
	if len(mj) == 0 && len(md) == 0 && len(nj) == 0 && len(nd) == 0 {
		return applier.upgrade()
	}
	return applier.scaleCluster(mj, md, nj, nd)
}

func (applier *Applier) scaleCluster(mj, md, nj, nd []net.IP) error {
	logrus.Info("Start to scale this cluster")
	logrus.Debugf("current cluster: master %s, worker %s", applier.ClusterCurrent.GetMasterIPList(), applier.ClusterCurrent.GetNodeIPList())

	scaleProcessor, err := processor.NewScaleProcessor(applier.ClusterFile.GetKubeadmConfig(), applier.ClusterFile, mj, md, nj, nd)
	if err != nil {
		return err
	}
	var cluster *v2.Cluster
	if !scaleProcessor.(*processor.ScaleProcessor).IsScaleUp {
		c, err := utils.DecodeCRDFromFile(common.GetClusterWorkClusterfile(applier.ClusterDesired.Name), common.Cluster)
		if err != nil {
			return err
		} else if c != nil {
			cluster = c.(*v2.Cluster)
		}
	} else {
		cluster = applier.ClusterDesired
	}
	err = processor.NewExecutor(scaleProcessor).Execute(cluster)
	if err != nil {
		return err
	}

	logrus.Info("Succeeded in scaling this cluster")

	return nil
}

func (applier *Applier) Upgrade(upgradeImgName string) error {
	if err := applier.initClusterfile(); err != nil {
		return err
	}
	if err := applier.initK8sClient(); err != nil {
		return err
	}

	applier.ClusterDesired.Spec.Image = upgradeImgName
	if err := applier.mountClusterImage(); err != nil {
		return err
	}
	defer func() {
		if err := applier.unMountClusterImage(); err != nil {
			logrus.Warnf("failed to umount cluster(%s): %v", applier.ClusterDesired.Name, err)
		}
	}()
	return applier.upgrade()
}

func (applier *Applier) upgrade() error {
	runtimeInterface, err := processor.ChooseRuntime(platform.DefaultMountClusterImageDir(applier.ClusterDesired.Name), applier.ClusterDesired, applier.ClusterFile.GetKubeadmConfig())
	if err != nil {
		return fmt.Errorf("failed to init runtime: %v", err)
	}
	upgradeImgMeta, err := runtimeInterface.GetClusterMetadata()
	if err != nil {
		return fmt.Errorf("failed to get cluster metadata: %v", err)
	}

	if applier.CurrentClusterInfo.GitVersion == upgradeImgMeta.Version {
		logrus.Infof("No upgrade required, image version and cluster version are both %s.", applier.CurrentClusterInfo.GitVersion)
		return nil
	}
	logrus.Infof("Start to upgrade this cluster from version(%s) to version(%s)", applier.CurrentClusterInfo.GitVersion, upgradeImgMeta.Version)

	upgradeProcessor, err := processor.NewUpgradeProcessor(platform.DefaultMountClusterImageDir(applier.ClusterDesired.Name), runtimeInterface)
	if err != nil {
		return err
	}
	err = upgradeProcessor.Execute(applier.ClusterDesired)
	if err != nil {
		return err
	}
	logrus.Infof("Succeeded in upgrading current cluster from version(%s) to version(%s)", applier.CurrentClusterInfo.GitVersion, upgradeImgMeta.Version)
	return clusterfile.SaveToDisk(applier.ClusterDesired, applier.ClusterDesired.Name)
}

func (applier *Applier) initClusterfile() (err error) {
	if applier.ClusterFile != nil {
		return nil
	}
	applier.ClusterFile, err = clusterfile.NewClusterFile(applier.ClusterDesired.GetAnnotationsByKey(common.ClusterfileName))
	return err
}

func (applier *Applier) initK8sClient() error {
	client, err := k8s.Newk8sClient()
	applier.Client = client
	if err != nil {
		return err
	}
	info, err := client.GetClusterVersion()
	applier.CurrentClusterInfo = info
	return err
}

func (applier *Applier) installApp() error {
	rootfs := platform.DefaultMountClusterImageDir(applier.ClusterDesired.Name)
	// use k8sClient to fetch current cluster version.
	info := applier.CurrentClusterInfo

	clusterMetadata, err := runtime.LoadMetadata(rootfs)
	if err != nil {
		return err
	}
	if clusterMetadata != nil {
		if !VersionCompatible(info.GitVersion, clusterMetadata.KubeVersion) {
			return fmt.Errorf("incompatible application version, need: %s", clusterMetadata.KubeVersion)
		}
	}

	installProcessor, err := processor.NewInstallProcessor(applier.ClusterFile)
	if err != nil {
		return err
	}
	err = processor.NewExecutor(installProcessor).Execute(applier.ClusterDesired)
	if err != nil {
		return err
	}

	return nil
}

func (applier *Applier) initCluster() error {
	logrus.Infof("Start to create a new cluster: master %s, worker %s", applier.ClusterDesired.GetMasterIPList(), applier.ClusterDesired.GetNodeIPList())
	createProcessor, err := processor.NewCreateProcessor(applier.ClusterFile)
	if err != nil {
		return err
	}

	if err := processor.NewExecutor(createProcessor).Execute(applier.ClusterDesired); err != nil {
		return err
	}

	logrus.Info("Succeeded in creating a new cluster, enjoy it!")

	return nil
}

func (applier *Applier) deleteCluster() error {
	deleteProcessor, err := processor.NewDeleteProcessor(applier.ClusterFile)
	if err != nil {
		return err
	}
	if err := applier.mountClusterImage(); err != nil {
		return err
	}
	//deleteProcessor to unmount image
	if err := processor.NewExecutor(deleteProcessor).Execute(applier.ClusterDesired); err != nil {
		return err
	}

	logrus.Info("Succeeded in deleting current cluster")

	return nil
}
