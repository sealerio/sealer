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
	"io/ioutil"
	"path/filepath"

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"
	"github.com/sealerio/sealer/cmd/sealer/cmd/utils"
	"github.com/sealerio/sealer/common"
	clusterruntime "github.com/sealerio/sealer/pkg/cluster-runtime"
	"github.com/sealerio/sealer/pkg/clusterfile"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var ScaleUpFlags *types.Flags

var longScaleUpCmdDescription = `scale-up command is used to scale-up master or node to the existing cluster.
User can scale-up cluster by explicitly specifying host IP`

var exampleForScaleUpCmd = `
scale-up cluster:
  sealer join --masters 192.168.0.1 --nodes 192.168.0.2 -p Sealer123
  sealer join --masters 192.168.0.1-192.168.0.3 --nodes 192.168.0.4-192.168.0.6 -p Sealer123
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

			workClusterfile := common.GetDefaultClusterfile()
			clusterFileData, err := ioutil.ReadFile(filepath.Clean(workClusterfile))
			if err != nil {
				return err
			}

			cf, err = clusterfile.NewClusterFile(clusterFileData)
			if err != nil {
				return err
			}

			//store the Cluster as CfSnapshot for rollback
			cf.CommitSnapshot()

			cluster := cf.GetCluster()
			if err = utils.ConstructClusterForScaleUp(&cluster, ScaleUpFlags, scaleUpMasterIPList, scaleUpNodeIPList); err != nil {
				return err
			}
			cf.SetCluster(cluster)

			//save desired clusterfile
			if err = cf.SaveAll(); err != nil {
				return err
			}

			defer func() {
				if err == nil {
					return
				}
				//if there exists an error,rollback the ClusterFile to the default file
				cf.RollBackClusterFile()
			}()

			infraDriver, err := infradriver.NewInfraDriver(&cluster)
			if err != nil {
				return err
			}

			var (
				clusterImageName = cluster.Spec.Image
				newHosts         = append(scaleUpMasterIPList, scaleUpNodeIPList...)
			)

			imageEngine, err := imageengine.NewImageEngine(imagecommon.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

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
				if e := imageMounter.Umount(clusterImageName, imageMountInfo); e != nil {
					logrus.Errorf("failed to umount cluster image: %v", e)
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
			_, _, err = installer.ScaleUp(scaleUpMasterIPList, scaleUpNodeIPList)
			if err != nil {
				return err
			}

			if err = cf.SaveAll(); err != nil {
				return err
			}

			return nil
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
