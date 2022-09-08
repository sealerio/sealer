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

package cmd

import (
	cluster_runtime "github.com/sealerio/sealer/pkg/cluster-runtime"
	"github.com/sealerio/sealer/pkg/infradriver"
	"os"
	"path/filepath"

	"github.com/sealerio/sealer/apply"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/utils/strings"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var runArgs *apply.Args

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "start to run a cluster from a ClusterImage",
	Long:  `sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8 --masters [arg] --nodes [arg]`,
	Example: `
create cluster to your bare metal server, appoint the iplist:
	sealer run kubernetes:v1.19.8 --masters 192.168.0.2,192.168.0.3,192.168.0.4 \
		--nodes 192.168.0.5,192.168.0.6,192.168.0.7 --passwd xxx

specify server SSH port :
  All servers use the same SSH port (default port: 22):
	sealer run kubernetes:v1.19.8 --masters 192.168.0.2,192.168.0.3,192.168.0.4 \
	--nodes 192.168.0.5,192.168.0.6,192.168.0.7 --port 24 --passwd xxx

  Different SSH port numbers exist:
	sealer run kubernetes:v1.19.8 --masters 192.168.0.2,192.168.0.3:23,192.168.0.4:24 \
	--nodes 192.168.0.5:25,192.168.0.6:25,192.168.0.7:27 --passwd xxx

create a cluster with custom environment variables:
	sealer run -e DashBoardPort=8443 mydashboard:latest  --masters 192.168.0.2,192.168.0.3,192.168.0.4 \
	--nodes 192.168.0.5,192.168.0.6,192.168.0.7 --passwd xxx
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: remove this now, maybe we can support it later
		// set local ip address as master0 default ip if user input is empty.
		// this is convenient to execute `sealer run` without set many arguments.
		// Example looks like "sealer run kubernetes:v1.19.8"
		//if runArgs.Masters == "" {
		//	ip, err := net.GetLocalDefaultIP()
		//	if err != nil {
		//		return err
		//	}
		//	runArgs.Masters = ip
		//}

		var cf clusterfile.Interface
		if clusterFile != "" {
			var err error
			cf, err = clusterfile.NewClusterFile(clusterFile)
			if err != nil {
				return err
			}
		}

		cluster := cf.GetCluster()
		infraDriver, err := infradriver.NewInfraDriver(&cluster)
		if err != nil {
			return err
		}

		//TODO mount image and copy to cluster
		installer, err := cluster_runtime.NewInstaller(infraDriver, cf)
		if err != nil {
			return err
		}

		regDriver, kubeDriver, err := installer.Install()
		if err != nil {
			return err
		}

		// TODO install APP

		return nil
	},
}

func init() {
	runArgs = &apply.Args{}
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&clusterFile, "Clusterfile", "f", "Clusterfile", "Clusterfile path to apply a Kubernetes cluster")
	runCmd.Flags().StringVarP(&runArgs.Provider, "provider", "", "", "set infra provider, example `ALI_CLOUD`, the local server need ignore this")
	runCmd.Flags().StringVarP(&runArgs.Masters, "masters", "m", "", "set count or IPList to masters")
	runCmd.Flags().StringVarP(&runArgs.Nodes, "nodes", "n", "", "set count or IPList to nodes")
	runCmd.Flags().StringVar(&runArgs.ClusterName, "cluster-name", "my-cluster", "set cluster name")
	runCmd.Flags().StringVarP(&runArgs.User, "user", "u", "root", "set baremetal server username")
	runCmd.Flags().StringVarP(&runArgs.Password, "passwd", "p", "", "set cloud provider or baremetal server password")
	runCmd.Flags().Uint16Var(&runArgs.Port, "port", 22, "set the sshd service port number for the server (default port: 22)")
	runCmd.Flags().StringVar(&runArgs.Pk, "pk", filepath.Join(common.GetHomeDir(), ".ssh", "id_rsa"), "set baremetal server private key")
	runCmd.Flags().StringVar(&runArgs.PkPassword, "pk-passwd", "", "set baremetal server private key password")
	runCmd.Flags().StringSliceVar(&runArgs.CMDArgs, "cmd-args", []string{}, "set args for image cmd instruction")
	runCmd.Flags().StringSliceVarP(&runArgs.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	err := runCmd.RegisterFlagCompletionFunc("provider", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return strings.ContainPartial([]string{common.BAREMETAL, common.AliCloud, common.CONTAINER}, toComplete), cobra.ShellCompDirectiveNoFileComp
	})
	if err != nil {
		logrus.Errorf("provide completion for provider flag, err: %v", err)
		os.Exit(1)
	}
}
