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

	"github.com/sealerio/sealer/apply"
	"github.com/sealerio/sealer/cmd/sealer/cmd/utils"
	"github.com/sealerio/sealer/common"
	clusterruntime "github.com/sealerio/sealer/pkg/cluster-runtime"
	"github.com/sealerio/sealer/pkg/clusterfile"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes"
	utilsnet "github.com/sealerio/sealer/utils/net"
	"github.com/sealerio/sealer/utils/os/fs"

	"github.com/spf13/cobra"
)

var (
	deleteArgs        *apply.Args
	deleteClusterFile string
	deleteClusterName string
	mastersToDelete   []net.IP
	workersToDelete   []net.IP
	deleteAll         bool
)

var longDeleteCmdDescription = `delete command is used to delete part or all of existing cluster.
User can delete cluster by explicitly specifying node IP, Clusterfile, or cluster name.`

var exampleForDeleteCmd = `
delete default cluster: 
	sealer delete --masters x.x.x.x --nodes x.x.x.x
	sealer delete --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y
delete all:
	sealer delete --all [--force]
	sealer delete -f /root/.sealer/mycluster/Clusterfile [--force]
	sealer delete -c my-cluster [--force]
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
			workClusterfile := common.GetClusterWorkClusterfile()
			if deleteAll {
				return deleteCluster(workClusterfile)
			}
			return scaleDownCluster(workClusterfile)
		},
	}

	deleteArgs = &apply.Args{}
	deleteCmd.Flags().IPSliceVarP(&mastersToDelete, "masters", "m", nil, "reduce Count or IPList to masters")
	deleteCmd.Flags().IPSliceVarP(&workersToDelete, "nodes", "n", nil, "reduce Count or IPList to nodes")
	deleteCmd.Flags().StringVarP(&deleteClusterFile, "Clusterfile", "f", "", "delete a kubernetes cluster with Clusterfile Annotations")
	deleteCmd.Flags().StringVarP(&deleteClusterName, "cluster", "c", "", "delete a kubernetes cluster with cluster name")
	deleteCmd.Flags().StringSliceVarP(&deleteArgs.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	deleteCmd.Flags().BoolVar(&kubernetes.ForceDelete, "force", false, "We also can input an --force flag to delete cluster by force")
	deleteCmd.Flags().BoolVarP(&deleteAll, "all", "a", false, "this flags is for delete nodes, if this is true, empty all node ip")

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

	distributor, err := imagedistributor.NewScpDistributor(imageEngine, infraDriver)
	if err != nil {
		return nil, nil, nil, err
	}

	return distributor, infraDriver, installer, nil
}

func deleteCluster(workClusterfile string) error {
	var err error
	clusterFileData, err := ioutil.ReadFile(filepath.Clean(workClusterfile))
	if err != nil {
		return err
	}
	cf, err := clusterfile.NewClusterFile(clusterFileData)
	if err != nil {
		return err
	}
	distributor, infraDriver, installer, err := getRuntimeInterfaces(cf)
	if err != nil {
		return err
	}
	if err = installer.UnInstall(); err != nil {
		return err
	}
	// exec clean.sh
	ips := infraDriver.GetHostIPList()
	clusterRootfsDir := infraDriver.GetClusterRootfs()
	cleanFile := fmt.Sprintf(common.DefaultClusterClearBashFile, clusterRootfsDir)
	for _, ip := range ips {
		if err := infraDriver.CmdAsync(ip, cleanFile); err != nil {
			return fmt.Errorf("failed to exec command(%s) on host(%s): error(%v)", cleanFile, ip, err)
		}
	}
	//delete rootfs file
	if err := distributor.Restore(infraDriver.GetClusterRootfs(), infraDriver.GetHostIPList()); err != nil {
		return err
	}

	//todo delete CleanFs
	if err := fs.FS.RemoveAll(common.GetClusterWorkDir(), common.DefaultClusterBaseDir(clusterName),
		common.DefaultKubeConfigDir()); err != nil {
		return err
	}
	return nil
}

func scaleDownCluster(workClusterfile string) error {
	clusterFileData, err := ioutil.ReadFile(filepath.Clean(workClusterfile))
	if err != nil {
		return err
	}
	cf, err := clusterfile.NewClusterFile(clusterFileData)
	if err != nil {
		return err
	}
	cluster := cf.GetCluster()

	hosts, err := utils.GetHosts(deleteArgs.Masters, deleteArgs.Nodes)
	if err != nil {
		return err
	}

	for _, host := range cluster.Spec.Hosts {
		for _, ip := range hosts {
			host.IPS = utilsnet.RemoveIPs(host.IPS, ip.IPS)
		}
	}

	cf.SetCluster(cluster)

	distributor, infraDriver, installer, err := getRuntimeInterfaces(cf)
	if err != nil {
		return err
	}
	for _, host := range hosts {
		if err := distributor.Restore(infraDriver.GetClusterRootfs(), host.IPS); err != nil {
			return err
		}
	}

	_, _, err = installer.ScaleDown(mastersToDelete, workersToDelete)
	if err != nil {
		return err
	}
	if err := cf.SaveAll(); err != nil {
		return err
	}
	return nil
}
