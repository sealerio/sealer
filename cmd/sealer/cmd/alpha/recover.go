// Copyright © 2022 Alibaba Group Holding Ltd.
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

package alpha

import (
	"fmt"
	"net"
	"path/filepath"
	"strconv"

	"github.com/sealerio/sealer/cmd/sealer/cmd/utils"
	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/pkg/infradriver"
	netutils "github.com/sealerio/sealer/utils/net"
	"github.com/sealerio/sealer/utils/shellcommand"
	"github.com/spf13/cobra"
)

type RecoverFlags struct {
	Host net.IP
}

var recoverFlag *RecoverFlags

var longRecoverCmdDescription = ` `

var exampleForRecoverCmd = `The following command will recover master0 node through specified host ip:
  sealer alpha recover master0 --host 192.168.1.203
`

// NewRecoverCmd recover master0
func NewRecoverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recover",
		Short: "sealer experimental sub-commands for recover cluster",
		Long:  longAlphaCmdDescription,
	}
	cmd.AddCommand(NewRecoverMaster0Cmd())
	return cmd
}

// NewRecoverMaster0Cmd recover master0
func NewRecoverMaster0Cmd() *cobra.Command {
	recoverCmd := &cobra.Command{
		Use:     "master0",
		Short:   "recover master0 to specified host by sealer",
		Long:    longRecoverCmdDescription,
		Example: exampleForRecoverCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			recoverHost := recoverFlag.Host
			if recoverHost == nil {
				return fmt.Errorf("recover host cannot be empty")
			}

			cf, err := clusterfile.NewClusterFile(nil)
			if err != nil {
				return err
			}

			cluster := cf.GetCluster()
			infraDriver, err := infradriver.NewInfraDriver(&cluster)
			if err != nil {
				return err
			}

			regConfig := infraDriver.GetClusterRegistryConfig()
			if !netutils.IsInIPList(recoverHost, regConfig.LocalRegistry.DeployHosts) {
				return fmt.Errorf("recover host(%s) must in local registry deploy host (%s)",
					recoverHost, regConfig.LocalRegistry.DeployHosts)
			}

			rootfs := infraDriver.GetClusterRootfsPath()
			dataDir := filepath.Join(rootfs, "registry")
			//todo use registry driver to launch registry
			initRegistry := fmt.Sprintf("cd %s/scripts && bash init-registry.sh %s %s %s", infraDriver.GetClusterRootfsPath(),
				strconv.Itoa(regConfig.LocalRegistry.Port), dataDir, regConfig.LocalRegistry.Domain)
			if err := infraDriver.CmdAsync(recoverHost, initRegistry); err != nil {
				return err
			}

			client := utils.GetClusterClient()
			if client == nil {
				return fmt.Errorf("failed to init cluster client")
			}

			currentCluster, err := utils.GetCurrentCluster(client)
			if err != nil {
				return fmt.Errorf("failed to get current cluster: %v", err)
			}
			// get current host ip list
			var hosts []net.IP
			for _, host := range currentCluster.Spec.Hosts {
				hosts = append(hosts, host.IPS...)
			}

			// remove old registry hosts alias
			uninstallCmd := shellcommand.CommandUnSetHostAlias(shellcommand.DefaultSealerHostAliasForRegistry)
			removeFunc := func(host net.IP) error {
				err := infraDriver.CmdAsync(host, uninstallCmd)
				if err != nil {
					return fmt.Errorf("failed to delete registry configuration: %v", err)
				}
				return nil
			}

			err = infraDriver.Execute(hosts, removeFunc)
			if err != nil {
				return err
			}

			// add new registry hosts alias
			addFunc := func(host net.IP) error {
				err := infraDriver.CmdAsync(host, shellcommand.CommandSetHostAlias(regConfig.LocalRegistry.Domain,
					recoverHost.String(), shellcommand.DefaultSealerHostAliasForRegistry))
				if err != nil {
					return fmt.Errorf("failed to config cluster hosts file cmd: %v", err)
				}
				return nil
			}

			err = infraDriver.Execute(hosts, addFunc)
			if err != nil {
				return err
			}

			return nil
		},
	}

	recoverFlag = &RecoverFlags{}
	recoverCmd.Flags().IPVar(&recoverFlag.Host, "host", nil, "ip address that the new master0 will be recovered on")
	return recoverCmd
}
