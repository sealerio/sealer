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
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	exampleForScaleUpCmd = `
scale-up cluster:
  sealer scale-up --masters 192.168.0.1 --nodes 192.168.0.2 -p 'Sealer123'
  sealer scale-up --masters 192.168.0.1-192.168.0.3 --nodes 192.168.0.4-192.168.0.6 -p 'Sealer123'
`
	longDescriptionForScaleUpCmd = `scale-up command is used to scale-up master or node to the existing cluster.
User can scale-up cluster by explicitly specifying host IP`

	exampleForJoinCmd = `
join cluster:
  sealer join --masters 192.168.0.1 --nodes 192.168.0.2 -p 'Sealer123'
  sealer join --masters 192.168.0.1-192.168.0.3 --nodes 192.168.0.4-192.168.0.6 -p 'Sealer123'
`
	longDescriptionForJoinCmd = `join command is used to join master or node to the existing cluster.
User can join cluster by explicitly specifying host IP`
)

func NewJoinCmd() *cobra.Command {
	joinFlags := &types.ScaleUpFlags{}
	joinCmd := &cobra.Command{
		Use:     "join",
		Short:   "join new master or worker node to specified cluster",
		Long:    longDescriptionForJoinCmd,
		Args:    cobra.NoArgs,
		Example: exampleForJoinCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			logrus.Warn("sealer join command will be deprecated in the future, please use sealer scale-up instead.")

			return scaleUpRunFunc(joinFlags)
		},
	}

	joinCmd.Flags().StringVarP(&joinFlags.User, "user", "u", "root", "set baremetal server username")
	joinCmd.Flags().StringVarP(&joinFlags.Password, "passwd", "p", "", "set cloud provider or baremetal server password")
	joinCmd.Flags().Uint16Var(&joinFlags.Port, "port", 22, "set the sshd service port number for the server (default port: 22)")
	joinCmd.Flags().StringVar(&joinFlags.Pk, "pk", filepath.Join(common.GetHomeDir(), ".ssh", "id_rsa"), "set baremetal server private key")
	joinCmd.Flags().StringVar(&joinFlags.PkPassword, "pk-passwd", "", "set baremetal server private key password")
	joinCmd.Flags().StringSliceVarP(&joinFlags.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	joinCmd.Flags().StringVarP(&joinFlags.Masters, "masters", "m", "", "set Count or IPList to masters")
	joinCmd.Flags().StringVarP(&joinFlags.Nodes, "nodes", "n", "", "set Count or IPList to nodes")
	joinCmd.Flags().BoolVar(&joinFlags.IgnoreCache, "ignore-cache", false, "whether ignore cache when distribute sealer image, default is false.")

	return joinCmd
}

func NewScaleUpCmd() *cobra.Command {
	scaleUpFlags := &types.ScaleUpFlags{}
	scaleUpFlagsCmd := &cobra.Command{
		Use:     "scale-up",
		Short:   "scale-up new master or worker node to specified cluster",
		Long:    longDescriptionForScaleUpCmd,
		Args:    cobra.NoArgs,
		Example: exampleForScaleUpCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			return scaleUpRunFunc(scaleUpFlags)
		},
	}

	scaleUpFlagsCmd.Flags().StringVarP(&scaleUpFlags.User, "user", "u", "root", "set baremetal server username")
	scaleUpFlagsCmd.Flags().StringVarP(&scaleUpFlags.Password, "passwd", "p", "", "set cloud provider or baremetal server password")
	scaleUpFlagsCmd.Flags().Uint16Var(&scaleUpFlags.Port, "port", 22, "set the sshd service port number for the server (default port: 22)")
	scaleUpFlagsCmd.Flags().StringVar(&scaleUpFlags.Pk, "pk", filepath.Join(common.GetHomeDir(), ".ssh", "id_rsa"), "set baremetal server private key")
	scaleUpFlagsCmd.Flags().StringVar(&scaleUpFlags.PkPassword, "pk-passwd", "", "set baremetal server private key password")
	scaleUpFlagsCmd.Flags().StringSliceVarP(&scaleUpFlags.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	scaleUpFlagsCmd.Flags().StringVarP(&scaleUpFlags.Masters, "masters", "m", "", "set Count or IPList to masters")
	scaleUpFlagsCmd.Flags().StringVarP(&scaleUpFlags.Nodes, "nodes", "n", "", "set Count or IPList to nodes")
	scaleUpFlagsCmd.Flags().BoolVar(&scaleUpFlags.IgnoreCache, "ignore-cache", false, "whether ignore cache when distribute sealer image, default is false.")

	return scaleUpFlagsCmd
}

func scaleUpRunFunc(scaleUpFlags *types.ScaleUpFlags) error {
	var (
		cf  clusterfile.Interface
		err error
	)

	if err = utils.ValidateScaleIPStr(scaleUpFlags.Masters, scaleUpFlags.Nodes); err != nil {
		return fmt.Errorf("failed to validate input run args: %v", err)
	}

	scaleUpMasterIPList, scaleUpNodeIPList, err := utils.ParseToNetIPList(scaleUpFlags.Masters, scaleUpFlags.Nodes)
	if err != nil {
		return fmt.Errorf("failed to parse ip string to net IP list: %v", err)
	}

	cf, _, err = clusterfile.GetActualClusterFile()
	if err != nil {
		return err
	}

	cluster := cf.GetCluster()
	client := utils.GetClusterClient()
	if client == nil {
		return fmt.Errorf("failed to get cluster client")
	}

	currentCluster, err := utils.GetCurrentCluster(client)
	if err != nil {
		return fmt.Errorf("failed to get current cluster: %v", err)
	}
	currentNodes := currentCluster.GetAllIPList()

	mj, nj, err := utils.ConstructClusterForScaleUp(&cluster, scaleUpFlags, currentNodes, scaleUpMasterIPList, scaleUpNodeIPList)
	if err != nil {
		return err
	}

	imageEngine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
	if err != nil {
		return err
	}

	id, err := imageEngine.Pull(&options.PullOptions{
		Quiet:      false,
		PullPolicy: "missing",
		Image:      cluster.Spec.Image,
		Platform:   "local",
	})
	if err != nil {
		return err
	}

	imageSpec, err := imageEngine.Inspect(&options.InspectOptions{ImageNameOrID: id})
	if err != nil {
		return fmt.Errorf("failed to get sealer image extension: %s", err)
	}

	// merge image extension
	mergedWithExt := utils.MergeClusterWithImageExtension(&cluster, imageSpec.ImageExtension)

	cf.SetCluster(*mergedWithExt)

	kubeInstaller, err := NewKubeInstaller(cf, imageEngine, imageSpec)
	if err != nil {
		return err
	}

	return kubeInstaller.ScaleUp(mj, nj, KubeScaleUpOptions{
		IgnoreCache: scaleUpFlags.IgnoreCache,
	})
}
