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
	"github.com/sealerio/sealer/apply"
	"github.com/sealerio/sealer/cmd/sealer/cmd/utils"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/cluster-runtime"
	"github.com/sealerio/sealer/pkg/clusterfile"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/utils/yaml"
	"github.com/spf13/cobra"
	"net"
	"strings"
)

var clusterName string
var joinArgs *apply.Args
var newMasters, newWorkers []net.IP

var exampleForJoinCmd = `
join default cluster:
	sealer join --masters x.x.x.x --nodes x.x.x.x
    sealer join --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y
`

func NewJoinCmd() *cobra.Command {
	joinCmd := &cobra.Command{
		Use:   "join",
		Short: "join new master or worker node to specified cluster",
		// TODO: add long description.
		Long:    "",
		Args:    cobra.NoArgs,
		Example: exampleForJoinCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				cf      clusterfile.Interface
				err     error
				Masters []string
				Nodes   []string
			)
			localClusterFile := common.GetClusterWorkClusterfile()
			file, err := clusterfile.NewClusterFile(localClusterFile)
			if err != nil {
				return err
			}
			cluster := file.GetCluster()
			if err := utils.ValidateJoinArgs(joinArgs); err != nil {
				return fmt.Errorf("failed to validate input run args: %v", err)
			}
			allMasters := append(Masters, runArgs.Masters, joinArgs.Masters)
			allNodes := append(Nodes, runArgs.Nodes, joinArgs.Nodes)
			masterList := strings.Join(allMasters, ",")
			nodeList := strings.Join(allNodes, ",")
			resultHosts, err := utils.GetHosts(masterList, nodeList)
			if err != nil {
				return err
			}
			cluster.Spec.Hosts = resultHosts
			if err := yaml.UnmarshalFile(localClusterFile, cluster); err != nil {
				return err
			}
			cf, err = clusterfile.NewClusterFile(localClusterFile)
			if err != nil {
				return err
			}

			cluster := cf.GetCluster()
			infraDriver, err := infradriver.NewInfraDriver(&cluster)
			if err != nil {
				return err
			}

			imageEngine, err := imageengine.NewImageEngine(imagecommon.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			distributor, err := imagedistributor.NewScpDistributor(imageEngine, infraDriver)
			if err != nil {
				return err
			}

			var (
				clusterImageName = cluster.Spec.Image
				hosts            = append(newMasters, newWorkers...)
			)

			// distribute rootfs
			if err = distributor.Distribute(clusterImageName, hosts); err != nil {
				return err
			}

			runtimeConfig := new(clusterruntime.RuntimeConfig)
			if cf.GetPlugins() != nil {
				runtimeConfig.Plugins = cf.GetPlugins()
			}

			if cf.GetKubeadmConfig() != nil {
				runtimeConfig.KubeadmConfig = *cf.GetKubeadmConfig()
			}

			installer, err := clusterruntime.NewInstaller(infraDriver, imageEngine, *runtimeConfig)

			_, _, err = installer.ScaleUp(newMasters, newWorkers)

			return err
		},
	}

	joinArgs = &apply.Args{}
	joinCmd.Flags().StringVarP(&joinArgs.User, "user", "u", "root", "set baremetal server username")
	joinCmd.Flags().StringVarP(&joinArgs.Password, "passwd", "p", "", "set cloud provider or baremetal server password")
	joinCmd.Flags().Uint16Var(&joinArgs.Port, "port", 22, "set the sshd service port number for the server (default port: 22)")
	joinCmd.Flags().StringVar(&joinArgs.Pk, "pk", common.GetHomeDir()+"/.ssh/id_rsa", "set baremetal server private key")
	joinCmd.Flags().StringVar(&joinArgs.PkPassword, "pk-passwd", "", "set baremetal server private key password")
	joinCmd.Flags().StringSliceVarP(&joinArgs.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	joinCmd.Flags().IPSliceVarP(&newMasters, "masters", "m", nil, "set Count or IPList to masters")
	joinCmd.Flags().IPSliceVarP(&newWorkers, "nodes", "n", nil, "set Count or IPList to nodes")
	joinCmd.Flags().StringVarP(&clusterName, "cluster-name", "c", "", "specify the name of cluster")
	return joinCmd
}
