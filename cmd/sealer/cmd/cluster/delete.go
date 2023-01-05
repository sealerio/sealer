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
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

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
	"github.com/sealerio/sealer/utils/os/fs"
)

var (
	deleteFlags *types.DeleteFlags
)

var longDeleteCmdDescription = `delete command is used to delete part or all of existing cluster.
User can delete cluster by explicitly specifying host IP`

var exampleForDeleteCmd = `
delete cluster node: 
  sealer delete --nodes 192.168.0.1 [--force]
  sealer delete --masters 192.168.0.1 --nodes 192.168.0.2 [--force]
  sealer delete --masters 192.168.0.1-192.168.0.3 --nodes 192.168.0.4-192.168.0.6 [--force]
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
				mastersToDelete    = deleteFlags.Masters
				workersToDelete    = deleteFlags.Nodes
				deleteAll          = deleteFlags.DeleteAll
				specifyClusterfile = deleteFlags.ClusterFile
				forceDelete        = deleteFlags.ForceDelete
			)

			if mastersToDelete == "" && workersToDelete == "" && !deleteAll {
				return fmt.Errorf("you must input node ip Or set flag -a")
			}

			if deleteAll {
				return deleteCluster(specifyClusterfile, forceDelete)
			}
			return scaleDownCluster(mastersToDelete, workersToDelete, forceDelete)
		},
	}

	deleteFlags = &types.DeleteFlags{}
	deleteCmd.Flags().StringVarP(&deleteFlags.Masters, "masters", "m", "", "reduce Count or IPList to masters")
	deleteCmd.Flags().StringVarP(&deleteFlags.Nodes, "nodes", "n", "", "reduce Count or IPList to nodes")
	deleteCmd.Flags().StringVarP(&deleteFlags.ClusterFile, "Clusterfile", "f", "", "delete a kubernetes cluster with Clusterfile")
	deleteCmd.Flags().StringSliceVarP(&deleteFlags.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	deleteCmd.Flags().BoolVar(&deleteFlags.ForceDelete, "force", false, "We also can input an --force flag to delete cluster by force")
	deleteCmd.Flags().BoolVarP(&deleteFlags.DeleteAll, "all", "a", false, "this flags is for delete the entire cluster, default is false")
	deleteCmd.Flags().BoolVarP(&imagedistributor.IsPrune, "prune", "p", true, "this flags is for delete all cluster rootfs, default is true")

	return deleteCmd
}

func deleteCluster(workClusterfile string, forceDelete bool) error {
	var (
		cf  clusterfile.Interface
		err error
	)

	if workClusterfile == "" {
		// use default clusterfile to do delete
		cf, err = clusterfile.NewClusterFile(nil)
		if err != nil {
			return err
		}
	} else {
		// use user specified clusterfile to do delete
		clusterFileData, err := os.ReadFile(filepath.Clean(workClusterfile))
		if err != nil {
			return err
		}

		cf, err = clusterfile.NewClusterFile(clusterFileData)
		if err != nil {
			return err
		}
	}

	//todo do we need CustomEnv ?
	//append custom env from CLI to cluster, if it is used to wrapper shell plugin
	cluster := cf.GetCluster()
	cluster.Spec.Env = append(cluster.Spec.Env, deleteFlags.CustomEnv...)

	if !forceDelete {
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
		err = imageMounter.Umount(cluster.Spec.Image, imageMountInfo)
		if err != nil {
			logrus.Errorf("failed to umount cluster image: %v", err)
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

func scaleDownCluster(masters, workers string, forceDelete bool) error {
	if err := cmdutils.ValidateScaleIPStr(masters, workers); err != nil {
		return fmt.Errorf("failed to validate input run args: %v", err)
	}

	deleteMasterIPList, deleteNodeIPList, err := cmdutils.ParseToNetIPList(masters, workers)
	if err != nil {
		return fmt.Errorf("failed to parse ip string to net IP list: %v", err)
	}

	cf, err := clusterfile.NewClusterFile(nil)
	if err != nil {
		return err
	}

	cluster := cf.GetCluster()
	//master0 machine cannot be deleted
	if cluster.Spec.Registry.LocalRegistry != nil && !*cluster.Spec.Registry.LocalRegistry.HA && netutils.IsInIPList(cluster.GetMaster0IP(), deleteMasterIPList) {
		return fmt.Errorf("master0 machine(%s) cannot be deleted when registry is noHA mode", cluster.GetMaster0IP())
	}
	// make sure deleted ip in current cluster
	var filteredDeleteMasterIPList []net.IP
	for _, ip := range deleteMasterIPList {
		if netutils.IsInIPList(ip, cluster.GetMasterIPList()) {
			filteredDeleteMasterIPList = append(filteredDeleteMasterIPList, ip)
		}
	}
	deleteMasterIPList = filteredDeleteMasterIPList

	var filteredDeleteNodeIPList []net.IP
	for _, ip := range deleteNodeIPList {
		// filter ip not in current cluster
		if netutils.IsInIPList(ip, cluster.GetNodeIPList()) {
			filteredDeleteNodeIPList = append(filteredDeleteNodeIPList, ip)
		}
	}
	deleteNodeIPList = filteredDeleteNodeIPList
	if len(deleteMasterIPList) == 0 && len(deleteNodeIPList) == 0 {
		logrus.Infof("both master and node need to be deleted all not in current cluster, skip delete")
		return nil
	}

	if !forceDelete {
		if err = confirmDeleteHosts(fmt.Sprintf("%s/%s", common.MASTER, common.NODE), append(deleteMasterIPList, deleteNodeIPList...)); err != nil {
			return err
		}
	}

	// TODO, env should be host env
	//cluster.Spec.Env = append(cluster.Spec.Env, deleteFlags.CustomEnv...)

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
		err = imageMounter.Umount(cluster.Spec.Image, imageMountInfo)
		if err != nil {
			logrus.Errorf("failed to umount cluster image: %v", err)
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

	if err = cf.SaveAll(clusterfile.SaveOptions{CommitToCluster: true}); err != nil {
		return err
	}
	return nil
}

func confirmDeleteHosts(role string, hostsToDelete []net.IP) error {
	if pass, err := utils.ConfirmOperation(fmt.Sprintf("Are you sure to delete these %s: %v? ", role, hostsToDelete)); err != nil {
		return err
	} else if !pass {
		return fmt.Errorf("exit the operation of delete these nodes")
	}

	return nil
}
