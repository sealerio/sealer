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
	"os"
	"path/filepath"

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"
	"github.com/sealerio/sealer/cmd/sealer/cmd/utils"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"
	imagev1 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var (
	exampleForRunCmd = `
run cluster by Clusterfile: 
  sealer run -f Clusterfile
run cluster by CLI flags:
  sealer run docker.io/sealerio/kubernetes:v1-22-15-sealerio-2 -m 172.16.130.21 -n 172.16.130.22 -p 'Sealer123'
run app image:
  sealer run localhost/nginx:v1
`

	longDescriptionForRunCmd = `sealer run docker.io/sealerio/kubernetes:v1.22.15 --masters [arg] --nodes [arg]`
)

func NewRunCmd() *cobra.Command {
	runFlags := &types.RunFlags{}
	runCmd := &cobra.Command{
		Use:     "run",
		Short:   "start to run a cluster from a sealer image",
		Long:    longDescriptionForRunCmd,
		Example: exampleForRunCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: remove this now, maybe we can support it later
			// set local ip address as master0 default ip if user input is empty.
			// this is convenient to execute `sealer run` without set many arguments.
			// Example looks like "sealer run docker.io/sealerio/kubernetes:v1.22.15"
			//if runFlags.Masters == "" {
			//	ip, err := net.GetLocalDefaultIP()
			//	if err != nil {
			//		return err
			//	}
			//	runFlags.Masters = ip
			//}
			var (
				err         error
				clusterFile = runFlags.ClusterFile
			)

			if len(args) == 0 && clusterFile == "" {
				return fmt.Errorf("you must input image name Or use Clusterfile")
			}

			if err = utils.ValidateRunHosts(runFlags.Masters, runFlags.Nodes); err != nil {
				return fmt.Errorf("failed to validate input run master or node: %v", err)
			}

			if clusterFile != "" {
				return runWithClusterfile(clusterFile, runFlags)
			}

			return runWithArgs(args[0], runFlags)
		},
	}

	//todo remove provider Flag now, maybe we can support it later
	//runCmd.Flags().StringVarP(&runFlags.Provider, "provider", "", "", "set infra provider, example `ALI_CLOUD`, the local server need ignore this")
	runCmd.Flags().StringVarP(&runFlags.Masters, "masters", "m", "", "set count or IPList to masters")
	runCmd.Flags().StringVarP(&runFlags.Nodes, "nodes", "n", "", "set count or IPList to nodes")
	runCmd.Flags().StringVarP(&runFlags.User, "user", "u", "root", "set baremetal server username")
	runCmd.Flags().StringVarP(&runFlags.Password, "passwd", "p", "", "set cloud provider or baremetal server password")
	runCmd.Flags().Uint16Var(&runFlags.Port, "port", 22, "set the sshd service port number for the server (default port: 22)")
	runCmd.Flags().StringVar(&runFlags.Pk, "pk", filepath.Join(common.GetHomeDir(), ".ssh", "id_rsa"), "set baremetal server private key")
	runCmd.Flags().StringVar(&runFlags.PkPassword, "pk-passwd", "", "set baremetal server private key password")
	runCmd.Flags().StringSliceVar(&runFlags.Cmds, "cmds", []string{}, "override default LaunchCmds of sealer image")
	runCmd.Flags().StringSliceVar(&runFlags.AppNames, "apps", nil, "override default AppNames of sealer image")
	runCmd.Flags().StringSliceVarP(&runFlags.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	runCmd.Flags().StringVarP(&runFlags.ClusterFile, "Clusterfile", "f", "", "Clusterfile path to run a Kubernetes cluster")
	runCmd.Flags().StringVar(&runFlags.Mode, "mode", common.ApplyModeApply, "load images to the specified registry in advance")
	runCmd.Flags().BoolVar(&runFlags.IgnoreCache, "ignore-cache", false, "whether ignore cache when distribute sealer image, default is false.")
	runCmd.Flags().StringVar(&runFlags.Distributor, "distributor", "sftp", "distribution method to use (sftp, p2p), default is sftp.")

	//err := runCmd.RegisterFlagCompletionFunc("provider", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	//	return strings.ContainPartial([]string{common.BAREMETAL, common.AliCloud, common.CONTAINER}, toComplete), cobra.ShellCompDirectiveNoFileComp
	//})
	//if err != nil {
	//	logrus.Errorf("provide completion for provider flag, err: %v", err)
	//	os.Exit(1)
	//}
	return runCmd
}

func runWithClusterfile(clusterFile string, runFlags *types.RunFlags) error {
	var p2p bool
	switch runFlags.Distributor {
	case "sftp":
		p2p = false
	case "p2p":
		p2p = true
	default:
		return fmt.Errorf("invalid distributor %s", runFlags.Distributor)
	}

	clusterFileData, err := os.ReadFile(filepath.Clean(clusterFile))
	if err != nil {
		return err
	}

	cf, err := clusterfile.NewClusterFile(clusterFileData)
	if err != nil {
		return err
	}

	cluster, err := utils.MergeClusterWithFlags(cf.GetCluster(), &types.MergeFlags{
		Masters:    runFlags.Masters,
		Nodes:      runFlags.Nodes,
		CustomEnv:  runFlags.CustomEnv,
		User:       runFlags.User,
		Password:   runFlags.Password,
		PkPassword: runFlags.PkPassword,
		Pk:         runFlags.Pk,
		Port:       runFlags.Port,
		Cmds:       runFlags.Cmds,
		AppNames:   runFlags.AppNames,
	})
	if err != nil {
		return fmt.Errorf("failed to merge cluster with run args: %v", err)
	}

	cf.SetCluster(*cluster)

	imageName := cluster.Spec.Image
	imageEngine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
	if err != nil {
		return err
	}

	id, err := imageEngine.Pull(&options.PullOptions{
		Quiet:      false,
		PullPolicy: "missing",
		Image:      imageName,
		Platform:   "local",
	})
	if err != nil {
		return err
	}

	imageSpec, err := imageEngine.Inspect(&options.InspectOptions{ImageNameOrID: id})
	if err != nil {
		return fmt.Errorf("failed to get sealer image extension: %s", err)
	}

	if imageSpec.ImageExtension.Type == imagev1.AppInstaller {
		appSpec := utils.ConstructApplication(cf.GetApplication(), cluster.Spec.CMD, cluster.Spec.APPNames, runFlags.CustomEnv)

		appInstaller, err := NewApplicationInstaller(appSpec, imageSpec.ImageExtension, imageEngine)
		if err != nil {
			return err
		}

		return appInstaller.Install(imageName, AppInstallOptions{
			Envs:         runFlags.CustomEnv,
			RunMode:      runFlags.Mode,
			IgnoreCache:  runFlags.IgnoreCache,
			Distribution: types.P2PDistribution,
		})
	}

	kubeInstaller, err := NewKubeInstaller(cf, imageEngine, imageSpec)
	if err != nil {
		return err
	}

	return kubeInstaller.Install(imageName, KubeInstallOptions{
		RunMode:         runFlags.Mode,
		IgnoreCache:     runFlags.IgnoreCache,
		P2PDistribution: p2p,
	})
}

func runWithArgs(imageName string, runFlags *types.RunFlags) error {
	var p2p bool
	switch runFlags.Distributor {
	case "sftp":
		p2p = false
	case "p2p":
		p2p = true
	default:
		return fmt.Errorf("invalid distributor %s", runFlags.Distributor)
	}

	imageEngine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
	if err != nil {
		return err
	}

	id, err := imageEngine.Pull(&options.PullOptions{
		Quiet:      false,
		PullPolicy: "missing",
		Image:      imageName,
		Platform:   "local",
	})
	if err != nil {
		return err
	}

	imageSpec, err := imageEngine.Inspect(&options.InspectOptions{ImageNameOrID: id})
	if err != nil {
		return fmt.Errorf("failed to get sealer image extension: %s", err)
	}

	if imageSpec.ImageExtension.Type == imagev1.AppInstaller {
		appSpec := utils.ConstructApplication(nil, runFlags.Cmds, runFlags.AppNames, runFlags.CustomEnv)

		appInstaller, err := NewApplicationInstaller(appSpec, imageSpec.ImageExtension, imageEngine)
		if err != nil {
			return err
		}

		return appInstaller.Install(imageName, AppInstallOptions{
			Envs:         runFlags.CustomEnv,
			RunMode:      runFlags.Mode,
			IgnoreCache:  runFlags.IgnoreCache,
			Distribution: types.P2PDistribution,
		})
	}

	cluster, err := utils.ConstructClusterForRun(imageName, runFlags)
	if err != nil {
		return err
	}

	clusterData, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}

	cf, err := clusterfile.NewClusterFile(clusterData)
	if err != nil {
		return err
	}

	kubeInstaller, err := NewKubeInstaller(cf, imageEngine, imageSpec)
	if err != nil {
		return err
	}

	return kubeInstaller.Install(imageName, KubeInstallOptions{
		RunMode:         runFlags.Mode,
		IgnoreCache:     runFlags.IgnoreCache,
		P2PDistribution: p2p,
	})
}
