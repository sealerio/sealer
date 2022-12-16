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
	"net"
	"path/filepath"

	clusterruntime "github.com/sealerio/sealer/pkg/cluster-runtime"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"
	"github.com/sealerio/sealer/cmd/sealer/cmd/utils"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/spf13/cobra"
)

var ScaleUpFlags *types.Flags

var longScaleUpCmdDescription = `scale-up command is used to scale-up master or node to the existing cluster.
User can scale-up cluster by explicitly specifying host IP`

var exampleForScaleUpCmd = `
scale-up cluster:
  sealer scale-up --masters 192.168.0.1 --nodes 192.168.0.2 -p Sealer123
  sealer scale-up --masters 192.168.0.1-192.168.0.3 --nodes 192.168.0.4-192.168.0.6 -p Sealer123
`

func NewScaleUpCmd() *cobra.Command {
	scaleUpFlagsCmd := &cobra.Command{
		Use:     "scale-up",
		Short:   "scale-up new master or worker node to specified cluster",
		Long:    longScaleUpCmdDescription,
		Args:    cobra.NoArgs,
		Example: exampleForScaleUpCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				cf  clusterfile.Interface
				err error
			)

			if err = utils.ValidateScaleIPStr(ScaleUpFlags.Masters, ScaleUpFlags.Nodes); err != nil {
				return fmt.Errorf("failed to validate input run args: %v", err)
			}

			scaleUpMasterIPList, scaleUpNodeIPList, err := utils.ParseToNetIPList(ScaleUpFlags.Masters, ScaleUpFlags.Nodes)
			if err != nil {
				return fmt.Errorf("failed to parse ip string to net IP list: %v", err)
			}

			cf, err = clusterfile.NewClusterFile(nil)
			if err != nil {
				return err
			}

			cluster := cf.GetCluster()
			if err = utils.ConstructClusterForScaleUp(&cluster, ScaleUpFlags, scaleUpMasterIPList, scaleUpNodeIPList); err != nil {
				return err
			}
			cf.SetCluster(cluster)

			infraDriver, err := infradriver.NewInfraDriver(&cluster)
			if err != nil {
				return err
			}

			imageEngine, err := imageengine.NewImageEngine(imagecommon.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			return scaleUpCluster(cluster.Spec.Image, scaleUpMasterIPList, scaleUpNodeIPList, infraDriver, imageEngine, cf)
		},
	}

	ScaleUpFlags = &types.Flags{}
	scaleUpFlagsCmd.Flags().StringVarP(&ScaleUpFlags.User, "user", "u", "root", "set baremetal server username")
	scaleUpFlagsCmd.Flags().StringVarP(&ScaleUpFlags.Password, "passwd", "p", "", "set cloud provider or baremetal server password")
	scaleUpFlagsCmd.Flags().Uint16Var(&ScaleUpFlags.Port, "port", 22, "set the sshd service port number for the server (default port: 22)")
	scaleUpFlagsCmd.Flags().StringVar(&ScaleUpFlags.Pk, "pk", filepath.Join(common.GetHomeDir(), ".ssh", "id_rsa"), "set baremetal server private key")
	scaleUpFlagsCmd.Flags().StringVar(&ScaleUpFlags.PkPassword, "pk-passwd", "", "set baremetal server private key password")
	scaleUpFlagsCmd.Flags().StringSliceVarP(&ScaleUpFlags.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	scaleUpFlagsCmd.Flags().StringVarP(&ScaleUpFlags.Masters, "masters", "m", "", "set Count or IPList to masters")
	scaleUpFlagsCmd.Flags().StringVarP(&ScaleUpFlags.Nodes, "nodes", "n", "", "set Count or IPList to nodes")
	return scaleUpFlagsCmd
}

func scaleUpCluster(clusterImageName string, scaleUpMasterIPList, scaleUpNodeIPList []net.IP, infraDriver infradriver.InfraDriver, imageEngine imageengine.Interface, cf clusterfile.Interface) error {
	logrus.Infof("start to scale up cluster")

	var (
		newHosts = append(scaleUpMasterIPList, scaleUpNodeIPList...)
	)

	clusterHostsPlatform, err := infraDriver.GetHostsPlatform(newHosts)
	if err != nil {
		return err
	}

	imageMounter, err := imagedistributor.NewImageMounter(imageEngine, clusterHostsPlatform)
	if err != nil {
		return err
	}

	imageMountInfo, err := imageMounter.Mount(clusterImageName)
	if err != nil {
		return err
	}
	defer func() {
		err = imageMounter.Umount(clusterImageName, imageMountInfo)
		if err != nil {
			logrus.Errorf("failed to umount cluster image")
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
		Distributor: distributor,
		Plugins:     plugins,
	}

	if cf.GetKubeadmConfig() != nil {
		runtimeConfig.KubeadmConfig = *cf.GetKubeadmConfig()
	}

	installer, err := clusterruntime.NewInstaller(infraDriver, *runtimeConfig)
	if err != nil {
		return err
	}

	//we need to save desired clusterfile to local disk temporarily.
	//and will use it later to clean the cluster node if ScaleUp failed.
	if err = cf.SaveAll(clusterfile.SaveOptions{}); err != nil {
		return err
	}

	_, _, err = installer.ScaleUp(scaleUpMasterIPList, scaleUpNodeIPList)
	if err != nil {
		return err
	}

	if err = cf.SaveAll(clusterfile.SaveOptions{CommitToCluster: true}); err != nil {
		return err
	}

	logrus.Infof("succeeded in scaling up cluster")

	return nil
}
