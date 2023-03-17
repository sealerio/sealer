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

package cluster

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"
	"github.com/sealerio/sealer/cmd/sealer/cmd/utils"
	"github.com/sealerio/sealer/pkg/application"
	clusterruntime "github.com/sealerio/sealer/pkg/cluster-runtime"
	"github.com/sealerio/sealer/pkg/clusterfile"
	imagev1 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var upgradeFlags *types.UpgradeFlags

var longUpgradeCmdDescription = `upgrade command is used to upgrade a Kubernetes cluster via specified Clusterfile.`

var exampleForUpgradeCmd = `
  sealer upgrade docker.io/sealerio/kubernetes:v1.22.15-upgrade
`

func NewUpgradeCmd() *cobra.Command {
	upgradeCmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "upgrade a Kubernetes cluster via specified Clusterfile",
		Long:    longUpgradeCmdDescription,
		Example: exampleForUpgradeCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				err         error
				clusterFile = upgradeFlags.ClusterFile
			)
			if len(args) == 0 && clusterFile == "" {
				return fmt.Errorf("you must input image name Or use Clusterfile")
			}

			if clusterFile != "" {
				return upgradeWithClusterfile(clusterFile)
			}

			imageEngine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			id, err := imageEngine.Pull(&options.PullOptions{
				Quiet:      false,
				PullPolicy: "missing",
				Image:      args[0],
				Platform:   "local",
			})
			if err != nil {
				return err
			}

			imageSpec, err := imageEngine.Inspect(&options.InspectOptions{ImageNameOrID: id})
			if err != nil {
				return fmt.Errorf("failed to get sealer image extension: %s", err)
			}

			return upgradeCluster(imageEngine, imageSpec)
		},
	}

	upgradeFlags = &types.UpgradeFlags{}
	upgradeCmd.Flags().StringVarP(&upgradeFlags.ClusterFile, "Clusterfile", "f", "", "Clusterfile path to upgrade a Kubernetes cluster")

	return upgradeCmd
}

func upgradeCluster(imageEngine imageengine.Interface, imageSpec *imagev1.ImageSpec) error {
	if imageSpec.ImageExtension.Type != imagev1.KubeInstaller {
		return fmt.Errorf("exit upgrade process, wrong sealer image type: %s", imageSpec.ImageExtension.Type)
	}

	//get origin cluster
	cf, _, err := clusterfile.GetActualClusterFile()
	if err != nil {
		return err
	}
	cluster := cf.GetCluster()

	//update image of cluster
	cluster.Spec.Image = imageSpec.Name
	clusterData, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}

	//generate new cluster
	cf, err = clusterfile.NewClusterFile(clusterData)
	if err != nil {
		return err
	}
	cluster = cf.GetCluster()

	infraDriver, err := infradriver.NewInfraDriver(&cluster)
	if err != nil {
		return err
	}
	clusterHosts := infraDriver.GetHostIPList()

	clusterHostsPlatform, err := infraDriver.GetHostsPlatform(clusterHosts)
	if err != nil {
		return err
	}

	logrus.Infof("start to upgrade cluster with image: %s", imageSpec.Name)

	imageMounter, err := imagedistributor.NewImageMounter(imageEngine, clusterHostsPlatform)
	if err != nil {
		return err
	}

	imageMountInfo, err := imageMounter.Mount(imageSpec.Name)
	if err != nil {
		return err
	}

	defer func() {
		err = imageMounter.Umount(imageSpec.Name, imageMountInfo)
		if err != nil {
			logrus.Errorf("failed to umount sealer image")
		}
	}()

	distributor, err := imagedistributor.NewScpDistributor(imageMountInfo, infraDriver, cf.GetConfigs())
	if err != nil {
		return err
	}

	plugins, err := loadPluginsFromImage(imageMountInfo)
	if err != nil {
		return err
	}

	if cf.GetPlugins() != nil {
		plugins = append(plugins, cf.GetPlugins()...)
	}

	runtimeConfig := &clusterruntime.RuntimeConfig{
		Distributor:            distributor,
		Plugins:                plugins,
		ContainerRuntimeConfig: cluster.Spec.ContainerRuntime,
	}

	upgrader, err := clusterruntime.NewInstaller(infraDriver, *runtimeConfig, clusterruntime.GetClusterInstallInfo(imageSpec.ImageExtension.Labels, runtimeConfig.ContainerRuntimeConfig))
	if err != nil {
		return err
	}

	//we need to save desired clusterfile to local disk temporarily
	//and will use it later to clean the cluster node if apply failed.
	if err = cf.SaveAll(clusterfile.SaveOptions{}); err != nil {
		return err
	}

	err = upgrader.Upgrade()
	if err != nil {
		return err
	}

	confPath := clusterruntime.GetClusterConfPath(imageSpec.ImageExtension.Labels)
	logrus.Info(confPath)
	cmds := infraDriver.GetClusterLaunchCmds()
	appNames := infraDriver.GetClusterLaunchApps()

	// merge to application between v2.ClusterSpec, v2.Application and image extension
	v2App, err := application.NewV2Application(utils.ConstructApplication(cf.GetApplication(), cmds, appNames), imageSpec.ImageExtension)
	if err != nil {
		return fmt.Errorf("failed to parse application from Clusterfile:%v ", err)
	}

	// install application
	if err = v2App.Launch(infraDriver); err != nil {
		return err
	}
	if err = v2App.Save(application.SaveOptions{}); err != nil {
		return err
	}

	//save and commit
	if err = cf.SaveAll(clusterfile.SaveOptions{CommitToCluster: true, ConfPath: confPath}); err != nil {
		return err
	}

	logrus.Infof("succeeded in upgrading cluster with image %s", imageSpec.Name)

	return nil
}

func upgradeWithClusterfile(clusterFile string) error {
	clusterFileData, err := os.ReadFile(filepath.Clean(clusterFile))
	if err != nil {
		return err
	}

	cf, err := clusterfile.NewClusterFile(clusterFileData)
	if err != nil {
		return err
	}

	cluster := cf.GetCluster()
	imageName := cluster.Spec.Image
	imageEngine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
	if err != nil {
		return err
	}

	id, err := imageEngine.Pull(&options.PullOptions{
		Quiet:      false,
		PullPolicy: "missing",
		Image:      imageName,
		Platform:   "local",
	})
	if err != nil {
		return err
	}

	imageSpec, err := imageEngine.Inspect(&options.InspectOptions{ImageNameOrID: id})
	if err != nil {
		return fmt.Errorf("failed to get sealer image extension: %s", err)
	}

	return upgradeCluster(imageEngine, imageSpec)
}
