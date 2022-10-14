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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"
	"github.com/sealerio/sealer/cmd/sealer/cmd/utils"
	"github.com/sealerio/sealer/common"
	clusterruntime "github.com/sealerio/sealer/pkg/cluster-runtime"
	"github.com/sealerio/sealer/pkg/clusterfile"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
)

var joinFlags *types.Flags

var longJoinCmdDescription = `join command is used to join master or node to the existing cluster.
User can join cluster by explicitly specifying host IP`

var exampleForJoinCmd = `
join cluster:
	sealer join --masters x.x.x.x --nodes x.x.x.x -p xxxx
    sealer join --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y -p xxx
`

func NewJoinCmd() *cobra.Command {
	joinCmd := &cobra.Command{
		Use:     "join",
		Short:   "join new master or worker node to specified cluster",
		Long:    longJoinCmdDescription,
		Args:    cobra.NoArgs,
		Example: exampleForJoinCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				cf  clusterfile.Interface
				err error
			)

			if err = utils.ValidateScaleIPStr(joinFlags.Masters, joinFlags.Nodes); err != nil {
				return fmt.Errorf("failed to validate input run args: %v", err)
			}

			joinMasterIPList, joinNodeIPList, err := utils.ParseToNetIPList(joinFlags.Masters, joinFlags.Nodes)
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

			cluster := cf.GetCluster()
			if err = utils.ConstructClusterForScaleUp(&cluster, joinFlags, joinMasterIPList, joinNodeIPList); err != nil {
				return err
			}

			cf.SetCluster(cluster)
			infraDriver, err := infradriver.NewInfraDriver(&cluster)
			if err != nil {
				return err
			}

			var (
				clusterImageName = cluster.Spec.Image
				newHosts         = append(joinMasterIPList, joinNodeIPList...)
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
				err = imageMounter.Umount(imageMountInfo)
				if err != nil {
					logrus.Errorf("failed to umount cluster image")
				}
			}()

			distributor, err := imagedistributor.NewScpDistributor(imageMountInfo, infraDriver, cf.GetConfigs())
			if err != nil {
				return err
			}

			runtimeConfig := &clusterruntime.RuntimeConfig{
				Distributor: distributor,
			}
			if cf.GetPlugins() != nil {
				runtimeConfig.Plugins = cf.GetPlugins()
			}

			if cf.GetKubeadmConfig() != nil {
				runtimeConfig.KubeadmConfig = *cf.GetKubeadmConfig()
			}

			installer, err := clusterruntime.NewInstaller(infraDriver, *runtimeConfig)
			if err != nil {
				return err
			}
			_, _, err = installer.ScaleUp(joinMasterIPList, joinNodeIPList)
			if err != nil {
				return err
			}

			if err = cf.SaveAll(); err != nil {
				return err
			}

			return nil
		},
	}

	joinFlags = &types.Flags{}
	joinCmd.Flags().StringVarP(&joinFlags.User, "user", "u", "root", "set baremetal server username")
	joinCmd.Flags().StringVarP(&joinFlags.Password, "passwd", "p", "", "set cloud provider or baremetal server password")
	joinCmd.Flags().Uint16Var(&joinFlags.Port, "port", 22, "set the sshd service port number for the server (default port: 22)")
	joinCmd.Flags().StringVar(&joinFlags.Pk, "pk", common.GetHomeDir()+"/.ssh/id_rsa", "set baremetal server private key")
	joinCmd.Flags().StringVar(&joinFlags.PkPassword, "pk-passwd", "", "set baremetal server private key password")
	joinCmd.Flags().StringSliceVarP(&joinFlags.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	joinCmd.Flags().StringVarP(&joinFlags.Masters, "masters", "m", "", "set Count or IPList to masters")
	joinCmd.Flags().StringVarP(&joinFlags.Nodes, "nodes", "n", "", "set Count or IPList to nodes")
	return joinCmd
}
