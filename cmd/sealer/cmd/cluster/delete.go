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

	"github.com/sealerio/sealer/utils/os"
	"github.com/sirupsen/logrus"

	netutils "github.com/sealerio/sealer/utils/net"

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"
	"github.com/sealerio/sealer/cmd/sealer/cmd/utils"
	"github.com/sealerio/sealer/common"
	clusterruntime "github.com/sealerio/sealer/pkg/cluster-runtime"
	"github.com/sealerio/sealer/pkg/clusterfile"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/utils/os/fs"

	"github.com/spf13/cobra"
)

var (
	deleteArgs      *types.Args
	mastersToDelete string
	workersToDelete string
	deleteAll       bool
)

var longDeleteCmdDescription = `delete command is used to delete part or all of existing cluster.
User can delete cluster by explicitly specifying host IP`

var exampleForDeleteCmd = `
delete cluster node: 
    sealer delete --nodes x.x.x.x [--force]
	sealer delete --masters x.x.x.x --nodes x.x.x.x [--force]
	sealer delete --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y [--force]
delete all:
	sealer delete --all [--force]
`

// NewDeleteCmd deleteCmd represents the delete command
func NewDeleteCmd() *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:     "delete",
		Short:   "delete an existing cluster",
		Long:    longDeleteCmdDescription,
		Args:    cobra.NoArgs,
		Example: exampleForDeleteCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			if mastersToDelete == "" && workersToDelete == "" && !deleteAll {
				return fmt.Errorf("you must input node ip Or set flag -a")
			}

			workClusterfile := common.GetDefaultClusterfile()
			if deleteAll {
				return deleteCluster(workClusterfile)
			}
			return scaleDownCluster(mastersToDelete, workersToDelete, workClusterfile)
		},
	}

	deleteArgs = &types.Args{}
	deleteCmd.Flags().StringVarP(&mastersToDelete, "masters", "m", "", "reduce Count or IPList to masters")
	deleteCmd.Flags().StringVarP(&workersToDelete, "nodes", "n", "", "reduce Count or IPList to nodes")
	deleteCmd.Flags().StringSliceVarP(&deleteArgs.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	deleteCmd.Flags().BoolVar(&clusterruntime.ForceDelete, "force", false, "We also can input an --force flag to delete cluster by force")
	deleteCmd.Flags().BoolVarP(&deleteAll, "all", "a", false, "this flags is for delete the entire cluster, default is false")

	return deleteCmd
}

func getRuntimeInterfaces(cf clusterfile.Interface) (imagedistributor.Interface, infradriver.InfraDriver, *clusterruntime.Installer, error) {
	cluster := cf.GetCluster()
	infraDriver, err := infradriver.NewInfraDriver(&cluster)
	if err != nil {
		return nil, nil, nil, err
	}

	runtimeConfig := new(clusterruntime.RuntimeConfig)
	if cf.GetPlugins() != nil {
		runtimeConfig.Plugins = cf.GetPlugins()
	}

	if cf.GetKubeadmConfig() != nil {
		runtimeConfig.KubeadmConfig = *cf.GetKubeadmConfig()
	}

	installer, err := clusterruntime.NewInstaller(infraDriver, nil, *runtimeConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	imageEngine, err := imageengine.NewImageEngine(imagecommon.EngineGlobalConfigurations{})
	if err != nil {
		return nil, nil, nil, err
	}

	distributor, err := imagedistributor.NewScpDistributor(imageEngine, infraDriver, cf.GetConfigs())
	if err != nil {
		return nil, nil, nil, err
	}

	return distributor, infraDriver, installer, nil
}

func deleteCluster(workClusterfile string) error {
	if !os.IsFileExist(workClusterfile) {
		logrus.Info("no cluster found")
		return nil
	}
	clusterFileData, err := ioutil.ReadFile(filepath.Clean(workClusterfile))
	if err != nil {
		return err
	}

	cf, err := clusterfile.NewClusterFile(clusterFileData)
	if err != nil {
		return err
	}

	//todo do we need CustomEnv ?
	//append custom env from CLI to cluster, if it is used to wrapper shell plugin
	cluster := cf.GetCluster()
	cluster.Spec.Env = append(cluster.Spec.Env, deleteArgs.CustomEnv...)

	distributor, infraDriver, installer, err := getRuntimeInterfaces(cf)
	if err != nil {
		return err
	}

	if err = installer.UnInstall(); err != nil {
		return err
	}

	//Restore rootfs
	ips := infraDriver.GetHostIPList()
	clusterBaseDir := infraDriver.GetClusterBasePath()
	if err = distributor.Restore(clusterBaseDir, ips); err != nil {
		return err
	}

	//delete local files,including sealer workdir,cluster file under sealer,kubeconfig under home dir.
	if err = fs.FS.RemoveAll(common.GetSealerWorkDir(), common.DefaultClusterBaseDir(infraDriver.GetClusterName()),
		common.DefaultKubeConfigDir()); err != nil {
		return err
	}
	return nil
}

func scaleDownCluster(masters, workers, workClusterfile string) error {
	if err := utils.ValidateScaleIPStr(masters, workers); err != nil {
		return fmt.Errorf("failed to validate input run args: %v", err)
	}

	deleteMasterIPList, deleteNodeIPList, err := utils.ParseToNetIPList(masters, workers)
	if err != nil {
		return fmt.Errorf("failed to parse ip string to net IP list: %v", err)
	}

	clusterFileData, err := ioutil.ReadFile(filepath.Clean(workClusterfile))
	if err != nil {
		return err
	}

	cf, err := clusterfile.NewClusterFile(clusterFileData)
	if err != nil {
		return err
	}

	cluster := cf.GetCluster()
	//master0 machine cannot be deleted
	if !netutils.NotInIPList(cluster.GetMaster0IP(), deleteMasterIPList) {
		return fmt.Errorf("master0 machine(%s) cannot be deleted", cluster.GetMaster0IP())
	}
	cluster.Spec.Env = append(cluster.Spec.Env, deleteArgs.CustomEnv...)

	distributor, infraDriver, installer, err := getRuntimeInterfaces(cf)
	if err != nil {
		return err
	}

	_, _, err = installer.ScaleDown(deleteMasterIPList, deleteNodeIPList)
	if err != nil {
		return err
	}

	if err = distributor.Restore(infraDriver.GetClusterBasePath(), append(deleteMasterIPList, deleteNodeIPList...)); err != nil {
		return err
	}

	if err = utils.ConstructClusterForScaleDown(&cluster, deleteMasterIPList, deleteNodeIPList); err != nil {
		return err
	}
	cf.SetCluster(cluster)

	if err = cf.SaveAll(); err != nil {
		return err
	}
	return nil
}
