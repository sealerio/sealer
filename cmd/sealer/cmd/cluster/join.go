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
	"path/filepath"

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"
	"github.com/sealerio/sealer/cmd/sealer/cmd/utils"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var joinFlags *types.Flags

var longJoinCmdDescription = `join command is used to join master or node to the existing cluster.
User can join cluster by explicitly specifying host IP`

var exampleForJoinCmd = `
join cluster:
  sealer join --masters 192.168.0.1 --nodes 192.168.0.2 -p Sealer123
  sealer join --masters 192.168.0.1-192.168.0.3 --nodes 192.168.0.4-192.168.0.6 -p Sealer123
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
			logrus.Warn("sealer join command will be deprecated in the future, please use sealer scale-up instead.")

			if err = utils.ValidateScaleIPStr(joinFlags.Masters, joinFlags.Nodes); err != nil {
				return fmt.Errorf("failed to validate input run args: %v", err)
			}

			joinMasterIPList, joinNodeIPList, err := utils.ParseToNetIPList(joinFlags.Masters, joinFlags.Nodes)
			if err != nil {
				return fmt.Errorf("failed to parse ip string to net IP list: %v", err)
			}

			cf, err = clusterfile.NewClusterFile(nil)
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

			imageEngine, err := imageengine.NewImageEngine(imagecommon.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			return scaleUpCluster(cluster.Spec.Image, joinMasterIPList, joinNodeIPList, infraDriver, imageEngine, cf)
		},
	}

	joinFlags = &types.Flags{}
	joinCmd.Flags().StringVarP(&joinFlags.User, "user", "u", "root", "set baremetal server username")
	joinCmd.Flags().StringVarP(&joinFlags.Password, "passwd", "p", "", "set cloud provider or baremetal server password")
	joinCmd.Flags().Uint16Var(&joinFlags.Port, "port", 22, "set the sshd service port number for the server (default port: 22)")
	joinCmd.Flags().StringVar(&joinFlags.Pk, "pk", filepath.Join(common.GetHomeDir(), ".ssh", "id_rsa"), "set baremetal server private key")
	joinCmd.Flags().StringVar(&joinFlags.PkPassword, "pk-passwd", "", "set baremetal server private key password")
	joinCmd.Flags().StringSliceVarP(&joinFlags.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	joinCmd.Flags().StringVarP(&joinFlags.Masters, "masters", "m", "", "set Count or IPList to masters")
	joinCmd.Flags().StringVarP(&joinFlags.Nodes, "nodes", "n", "", "set Count or IPList to nodes")
	return joinCmd
}
