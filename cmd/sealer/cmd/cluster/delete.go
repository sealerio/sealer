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
	"net"
	"path/filepath"

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"
	cmdutils "github.com/sealerio/sealer/cmd/sealer/cmd/utils"
	"github.com/sealerio/sealer/common"
	clusterruntime "github.com/sealerio/sealer/pkg/cluster-runtime"
	"github.com/sealerio/sealer/pkg/clusterfile"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/utils"
	netutils "github.com/sealerio/sealer/utils/net"
	"github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/os/fs"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	deleteFlags        *types.Flags
	specifyClusterfile string
	deleteAll          bool
	ForceDelete        bool
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
			var (
				mastersToDelete = deleteFlags.Masters
				workersToDelete = deleteFlags.Nodes
			)
			workClusterfile := common.GetDefaultClusterfile()
			if mastersToDelete == "" && workersToDelete == "" && !deleteAll {
				return fmt.Errorf("you must input node ip Or set flag -a")
			}
			if specifyClusterfile != "" {
				workClusterfile = specifyClusterfile
			}
			if deleteAll {
				return deleteCluster(workClusterfile)
			}
			return scaleDownCluster(mastersToDelete, workersToDelete, workClusterfile)
		},
	}

	deleteFlags = &types.Flags{}
	deleteCmd.Flags().StringVarP(&deleteFlags.Masters, "masters", "m", "", "reduce Count or IPList to masters")
	deleteCmd.Flags().StringVarP(&deleteFlags.Nodes, "nodes", "n", "", "reduce Count or IPList to nodes")
	deleteCmd.Flags().StringVarP(&specifyClusterfile, "Clusterfile", "f", "", "delete a kubernetes cluster with Clusterfile")
	deleteCmd.Flags().StringSliceVarP(&deleteFlags.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	deleteCmd.Flags().BoolVar(&ForceDelete, "force", false, "We also can input an --force flag to delete cluster by force")
	deleteCmd.Flags().BoolVarP(&deleteAll, "all", "a", false, "this flags is for delete the entire cluster, default is false")
	deleteCmd.Flags().BoolVarP(&imagedistributor.IsPrune, "prune", "p", true, "this flags is for delete all cluster rootfs, default is true")

	return deleteCmd
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
	cluster.Spec.Env = append(cluster.Spec.Env, deleteFlags.CustomEnv...)

	if !ForceDelete {
		if err = confirmDeleteHosts(fmt.Sprintf("%s/%s", common.MASTER, common.NODE), cluster.GetAllIPList()); err != nil {
			return err
		}
	}

	infraDriver, err := infradriver.NewInfraDriver(&cluster)
	if err != nil {
		return err
	}

	imageEngine, err := imageengine.NewImageEngine(imagecommon.EngineGlobalConfigurations{})
	if err != nil {
		return err
	}

	clusterHostsPlatform, err := infraDriver.GetHostsPlatform(infraDriver.GetHostIPList())
	if err != nil {
		return err
	}

	imageMounter, err := imagedistributor.NewImageMounter(imageEngine, clusterHostsPlatform)
	if err != nil {
		return err
	}

	imageMountInfo, err := imageMounter.Mount(cluster.Spec.Image)
	if err != nil {
		return err
	}
	defer func() {
		err = imageMounter.Umount(imageMountInfo)
		if err != nil {
			logrus.Errorf("failed to umount cluster image")
		}
	}()

	distributor, err := imagedistributor.NewScpDistributor(imageMountInfo, infraDriver, nil)
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

	if err = installer.UnInstall(); err != nil {
		return err
	}

	//delete local files,including sealer workdir,cluster file under sealer,kubeconfig under home dir.
	if err = fs.FS.RemoveAll(common.GetSealerWorkDir(), common.DefaultKubeConfigDir()); err != nil {
		return err
	}

	// delete cluster file under sealer if isPrune is true
	if imagedistributor.IsPrune {
		if err = fs.FS.RemoveAll(common.DefaultClusterBaseDir(infraDriver.GetClusterName())); err != nil {
			return err
		}
	}

	return nil
}

func scaleDownCluster(masters, workers, workClusterfile string) error {
	if err := cmdutils.ValidateScaleIPStr(masters, workers); err != nil {
		return fmt.Errorf("failed to validate input run args: %v", err)
	}

	deleteMasterIPList, deleteNodeIPList, err := cmdutils.ParseToNetIPList(masters, workers)
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
	if netutils.IsInIPList(cluster.GetMaster0IP(), deleteMasterIPList) {
		return fmt.Errorf("master0 machine(%s) cannot be deleted", cluster.GetMaster0IP())
	}
	// make sure deleted ip in current cluster
	for _, ip := range deleteMasterIPList {
		if !netutils.IsInIPList(ip, cluster.GetMasterIPList()) {
			return fmt.Errorf("ip(%s) not found in current master list", ip)
		}
	}
	for _, ip := range deleteNodeIPList {
		if !netutils.IsInIPList(ip, cluster.GetNodeIPList()) {
			return fmt.Errorf("ip(%s) not found in current master list", ip)
		}
	}

	if !ForceDelete {
		if err = confirmDeleteHosts(fmt.Sprintf("%s/%s", common.MASTER, common.NODE), append(deleteMasterIPList, deleteNodeIPList...)); err != nil {
			return err
		}
	}

	cluster.Spec.Env = append(cluster.Spec.Env, deleteFlags.CustomEnv...)

	infraDriver, err := infradriver.NewInfraDriver(&cluster)
	if err != nil {
		return err
	}

	imageEngine, err := imageengine.NewImageEngine(imagecommon.EngineGlobalConfigurations{})
	if err != nil {
		return err
	}

	clusterHostsPlatform, err := infraDriver.GetHostsPlatform(append(deleteMasterIPList, deleteNodeIPList...))
	if err != nil {
		return err
	}

	imageMounter, err := imagedistributor.NewImageMounter(imageEngine, clusterHostsPlatform)
	if err != nil {
		return err
	}

	imageMountInfo, err := imageMounter.Mount(cluster.Spec.Image)
	if err != nil {
		return err
	}
	defer func() {
		err = imageMounter.Umount(imageMountInfo)
		if err != nil {
			logrus.Errorf("failed to umount cluster image")
		}
	}()

	distributor, err := imagedistributor.NewScpDistributor(imageMountInfo, infraDriver, nil)
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

	_, _, err = installer.ScaleDown(deleteMasterIPList, deleteNodeIPList)
	if err != nil {
		return err
	}

	if err = cmdutils.ConstructClusterForScaleDown(&cluster, deleteMasterIPList, deleteNodeIPList); err != nil {
		return err
	}
	cf.SetCluster(cluster)

	return cf.SaveAll()
}

func confirmDeleteHosts(role string, hostsToDelete []net.IP) error {
	if pass, err := utils.ConfirmOperation(fmt.Sprintf("Are you sure to delete these %s: %v? ", role, hostsToDelete)); err != nil {
		return err
	} else if !pass {
		return fmt.Errorf("exit the operation of delete these nodes")
	}

	return nil
}
