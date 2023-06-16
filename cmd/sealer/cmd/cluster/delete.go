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

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"
	cmdutils "github.com/sealerio/sealer/cmd/sealer/cmd/utils"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/utils"
	netutils "github.com/sealerio/sealer/utils/net"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	exampleForDeleteCmd = `
delete cluster node: 
  sealer delete --nodes 192.168.0.1 [--force]
  sealer delete --masters 192.168.0.1 --nodes 192.168.0.2 [--force]
  sealer delete --masters 192.168.0.1-192.168.0.3 --nodes 192.168.0.4-192.168.0.6 [--force]
delete all:
  sealer delete --all [--force]
  sealer delete -a -f Clusterfile [--force]
`

	longDescriptionForDeleteCmd = `delete command is used to delete part or all of existing cluster.
User can delete cluster by explicitly specifying host IP`
)

// NewDeleteCmd deleteCmd represents the delete command
func NewDeleteCmd() *cobra.Command {
	deleteFlags := &types.DeleteFlags{}
	deleteCmd := &cobra.Command{
		Use:     "delete",
		Short:   "delete an existing cluster",
		Long:    longDescriptionForDeleteCmd,
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
				return deleteCluster(specifyClusterfile, forceDelete, deleteFlags)
			}

			return scaleDownCluster(specifyClusterfile, mastersToDelete, workersToDelete, forceDelete, deleteFlags)
		},
	}

	deleteCmd.Flags().StringVarP(&deleteFlags.Masters, "masters", "m", "", "reduce Count or IPList to masters")
	deleteCmd.Flags().StringVarP(&deleteFlags.Nodes, "nodes", "n", "", "reduce Count or IPList to nodes")
	deleteCmd.Flags().StringVarP(&deleteFlags.ClusterFile, "Clusterfile", "f", "", "delete a kubernetes cluster with Clusterfile")
	deleteCmd.Flags().StringSliceVarP(&deleteFlags.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	deleteCmd.Flags().BoolVar(&deleteFlags.ForceDelete, "force", false, "We also can input an --force flag to delete cluster by force")
	deleteCmd.Flags().BoolVarP(&deleteFlags.DeleteAll, "all", "a", false, "this flags is for delete the entire cluster, default is false")
	deleteCmd.Flags().BoolVarP(&deleteFlags.Prune, "prune", "p", false, "this flags is for delete all cluster rootfs, default is false")

	return deleteCmd
}

func deleteCluster(workClusterfile string, forceDelete bool, deleteFlags *types.DeleteFlags) error {
	var (
		cf  clusterfile.Interface
		err error
	)

	if workClusterfile == "" {
		// use default clusterfile to do delete
		cf, _, err = clusterfile.GetActualClusterFile()
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
	cf.SetCluster(cluster)

	if !forceDelete {
		if err = confirmDeleteHosts(cluster.GetMasterIPList(), cluster.GetNodeIPList()); err != nil {
			return err
		}
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

	kubeInstaller, err := NewKubeInstaller(cf, imageEngine, imageSpec)
	if err != nil {
		return err
	}

	return kubeInstaller.Delete(KubeDeleteOptions{
		Prune: deleteFlags.Prune,
	})
}

func scaleDownCluster(workClusterfile, masters, workers string, forceDelete bool, deleteFlags *types.DeleteFlags) error {
	if err := cmdutils.ValidateScaleIPStr(masters, workers); err != nil {
		return fmt.Errorf("failed to validate input run args: %v", err)
	}

	deleteMasterIPList, deleteNodeIPList, err := cmdutils.ParseToNetIPList(masters, workers)
	if err != nil {
		return fmt.Errorf("failed to parse ip string to net IP list: %v", err)
	}

	cf, _, err := clusterfile.GetActualClusterFile()
	if err != nil {
		return err
	}
	if workClusterfile != "" {
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

	cluster := cf.GetCluster()
	//master0 machine cannot be deleted
	if cluster.Spec.Registry.LocalRegistry != nil && !*cluster.Spec.Registry.LocalRegistry.HA && netutils.IsInIPList(cluster.GetMaster0IP(), deleteMasterIPList) {
		return fmt.Errorf("master0 machine(%s) cannot be deleted when registry is noHA mode", cluster.GetMaster0IP())
	}
	// make sure deleted ip in current cluster
	var filteredDeleteMasterIPList []net.IP
	for _, ip := range deleteMasterIPList {
		if netutils.IsInIPList(ip, cluster.GetMasterIPList()) {
			if netutils.IsLocalIP(ip, nil) {
				return fmt.Errorf("not allow delete master %s from itself, you can login other master to execute this deletation", ip)
			}
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
		if err = confirmDeleteHosts(deleteMasterIPList, deleteNodeIPList); err != nil {
			return err
		}
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

	kubeInstaller, err := NewKubeInstaller(cf, imageEngine, imageSpec)
	if err != nil {
		return err
	}

	return kubeInstaller.ScaleDown(deleteMasterIPList, deleteNodeIPList, KubeScaleDownOptions{
		Prune: deleteFlags.Prune,
	})
}

func confirmDeleteHosts(masterToDelete, nodeToDelete []net.IP) error {
	prompt := "Are you sure to delete:"

	if len(masterToDelete) != 0 {
		prompt = fmt.Sprintf("%s %s %v", prompt, common.MASTER, masterToDelete)
	}

	if len(nodeToDelete) != 0 {
		prompt = fmt.Sprintf("%s %s %v", prompt, common.NODE, nodeToDelete)
	}

	if pass, err := utils.ConfirmOperation(prompt); err != nil {
		return err
	} else if !pass {
		return fmt.Errorf("exit the operation of delete these nodes")
	}

	return nil
}
