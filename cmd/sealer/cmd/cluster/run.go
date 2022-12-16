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
	"github.com/sealerio/sealer/cmd/sealer/cmd/utils"
	"github.com/sealerio/sealer/common"
	clusterruntime "github.com/sealerio/sealer/pkg/cluster-runtime"
	"github.com/sealerio/sealer/pkg/clusterfile"
	v12 "github.com/sealerio/sealer/pkg/define/image/v1"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sealerio/sealer/utils/platform"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var runFlags *types.Flags

var longNewRunCmdDescription = `sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8 --masters [arg] --nodes [arg]`

var exampleForRunCmd = `
run cluster by Clusterfile: 
  sealer run -f Clusterfile

run cluster by CLI flags:
  sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.22.4 -m 172.28.80.01 -n 172.28.80.02 -p Sealer123

run app image:
  sealer run localhost/nginx:v1
`

func NewRunCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:     "run",
		Short:   "start to run a cluster from a ClusterImage",
		Long:    longNewRunCmdDescription,
		Example: exampleForRunCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: remove this now, maybe we can support it later
			// set local ip address as master0 default ip if user input is empty.
			// this is convenient to execute `sealer run` without set many arguments.
			// Example looks like "sealer run kubernetes:v1.19.8"
			//if runFlags.Masters == "" {
			//	ip, err := net.GetLocalDefaultIP()
			//	if err != nil {
			//		return err
			//	}
			//	runFlags.Masters = ip
			//}
			var (
				cf              clusterfile.Interface
				clusterFileData []byte
				err             error
				clusterFile     = runFlags.ClusterFile
				applyMode       = runFlags.Mode
			)
			imageEngine, err := imageengine.NewImageEngine(imagecommon.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			extension, err := imageEngine.GetSealerImageExtension(&imagecommon.GetImageAnnoOptions{ImageNameOrID: args[0]})
			if err != nil {
				return fmt.Errorf("failed to get cluster image extension: %s", err)
			}

			if extension.Type == v12.AppInstaller {
				logrus.Infof("start to install app image: %s", args[0])
				cf, err := clusterfile.NewClusterFile(nil)
				if err != nil {
					return err
				}

				cluster := cf.GetCluster()
				infraDriver, err := infradriver.NewInfraDriver(&cluster)
				if err != nil {
					return err
				}

				if err := imageEngine.Pull(&imagecommon.PullOptions{
					Quiet:      false,
					PullPolicy: "missing",
					Image:      args[0],
					Platform:   "local",
				}); err != nil {
					return err
				}
				return installApplication(args[0],  runFlags.Cmds, runFlags.AppNames, extension, infraDriver, imageEngine, applyMode)
			}

			if len(runFlags.Cmds) > 0 {
				return fmt.Errorf("this command parameter (--cmds) is only available to application images")
			}

			if runFlags.Masters == "" && clusterFile == "" {
				return fmt.Errorf("you must input master ip Or use Clusterfile")
			}

			if clusterFile != "" {
				clusterFileData, err = os.ReadFile(filepath.Clean(clusterFile))
				if err != nil {
					return err
				}

				cf, err = clusterfile.NewClusterFile(clusterFileData)
				if err != nil {
					return err
				}
			} else {
				if len(args) == 0 {
					return fmt.Errorf("you must input cluster image name")
				}

				if err = utils.ValidateRunFlags(runFlags); err != nil {
					return fmt.Errorf("failed to validate input run args: %v", err)
				}

				cluster, err := utils.ConstructClusterForRun(args[0], runFlags)
				if err != nil {
					return err
				}

				clusterData, err := yaml.Marshal(cluster)
				if err != nil {
					return err
				}

				cf, err = clusterfile.NewClusterFile(clusterData)
				if err != nil {
					return err
				}
			}

			cluster := cf.GetCluster()
			infraDriver, err := infradriver.NewInfraDriver(&cluster)
			if err != nil {
				return err
			}

			return createNewCluster(infraDriver, imageEngine, cf, applyMode)
		},
	}
	runFlags = &types.Flags{}
	//todo remove provider Flag now, maybe we can support it later
	//runCmd.Flags().StringVarP(&runFlags.Provider, "provider", "", "", "set infra provider, example `ALI_CLOUD`, the local server need ignore this")
	runCmd.Flags().StringVarP(&runFlags.Masters, "masters", "m", "", "set count or IPList to masters")
	runCmd.Flags().StringVarP(&runFlags.Nodes, "nodes", "n", "", "set count or IPList to nodes")
	runCmd.Flags().StringVarP(&runFlags.User, "user", "u", "root", "set baremetal server username")
	runCmd.Flags().StringVarP(&runFlags.Password, "passwd", "p", "", "set cloud provider or baremetal server password")
	runCmd.Flags().Uint16Var(&runFlags.Port, "port", 22, "set the sshd service port number for the server (default port: 22)")
	runCmd.Flags().StringVar(&runFlags.Pk, "pk", filepath.Join(common.GetHomeDir(), ".ssh", "id_rsa"), "set baremetal server private key")
	runCmd.Flags().StringVar(&runFlags.PkPassword, "pk-passwd", "", "set baremetal server private key password")
	runCmd.Flags().StringSliceVar(&runFlags.Cmds, "cmds", []string{}, "override default LaunchCmds of clusterimage")
	runCmd.Flags().StringSliceVar(&runFlags.AppNames, "apps", []string{}, "override default AppNames of clusterimage")
	runCmd.Flags().StringSliceVarP(&runFlags.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	runCmd.Flags().StringVarP(&runFlags.ClusterFile, "Clusterfile", "f", "", "Clusterfile path to run a Kubernetes cluster")
	runCmd.Flags().StringVar(&runFlags.Mode, "mode", common.ApplyModeApply, "load images to the specified registry in advance")
	//err := runCmd.RegisterFlagCompletionFunc("provider", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	//	return strings.ContainPartial([]string{common.BAREMETAL, common.AliCloud, common.CONTAINER}, toComplete), cobra.ShellCompDirectiveNoFileComp
	//})
	//if err != nil {
	//	logrus.Errorf("provide completion for provider flag, err: %v", err)
	//	os.Exit(1)
	//}
	return runCmd
}

func createNewCluster(infraDriver infradriver.InfraDriver, imageEngine imageengine.Interface, cf clusterfile.Interface, mode string) error {
	var (
		clusterHosts     = infraDriver.GetHostIPList()
		clusterImageName = infraDriver.GetClusterImageName()
	)

	clusterHostsPlatform, err := infraDriver.GetHostsPlatform(clusterHosts)
	if err != nil {
		return err
	}

	imageMounter, err := imagedistributor.NewImageMounter(imageEngine, clusterHostsPlatform)
	if err != nil {
		return err
	}

	imageMountInfo, err := imageMounter.Mount(clusterImageName)
	if err != nil {
		return err
	}

	defer func() {
		err = imageMounter.Umount(clusterImageName, imageMountInfo)
		if err != nil {
			logrus.Errorf("failed to umount cluster image")
		}
	}()

	distributor, err := imagedistributor.NewScpDistributor(imageMountInfo, infraDriver, cf.GetConfigs())
	if err != nil {
		return err
	}

	if mode == common.ApplyModeLoadImage {
		return loadToRegistry(infraDriver, distributor)
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
		ImageEngine: imageEngine,
		Plugins:     plugins,
	}

	if cf.GetKubeadmConfig() != nil {
		runtimeConfig.KubeadmConfig = *cf.GetKubeadmConfig()
	}

	installer, err := clusterruntime.NewInstaller(infraDriver, *runtimeConfig)
	if err != nil {
		return err
	}

	//we need to save desired clusterfile to local disk temporarily
	//and will use it later to clean the cluster node if apply failed.
	if err = cf.SaveAll(clusterfile.SaveOptions{}); err != nil {
		return err
	}

	err = installer.Install()
	if err != nil {
		return err
	}

	//save and commit
	if err = cf.SaveAll(clusterfile.SaveOptions{CommitToCluster: true}); err != nil {
		return err
	}

	return nil
}

func loadPluginsFromImage(imageMountInfo []imagedistributor.ClusterImageMountInfo) (plugins []v1.Plugin, err error) {
	for _, info := range imageMountInfo {
		defaultPlatform := platform.GetDefaultPlatform()
		if info.Platform.ToString() == defaultPlatform.ToString() {
			plugins, err = clusterruntime.LoadPluginsFromFile(filepath.Join(info.MountDir, "plugins"))
			if err != nil {
				return
			}
		}
	}

	return plugins, nil
}

// loadToRegistry just load container image to local registry
func loadToRegistry(infraDriver infradriver.InfraDriver, distributor imagedistributor.Distributor) error {
	regConfig := infraDriver.GetClusterRegistryConfig()
	// todo only support load image to local registry at present
	if regConfig.LocalRegistry == nil {
		return nil
	}

	deployHosts := infraDriver.GetHostIPListByRole(common.MASTER)
	if len(deployHosts) < 1 {
		return fmt.Errorf("local registry host can not be nil")
	}
	master0 := deployHosts[0]

	logrus.Infof("start to apply with mode(%s)", common.ApplyModeLoadImage)
	if !regConfig.LocalRegistry.HaMode {
		deployHosts = []net.IP{master0}
	}

	if err := distributor.DistributeRegistry(deployHosts, filepath.Join(infraDriver.GetClusterRootfsPath(), "registry")); err != nil {
		return err
	}

	logrus.Infof("load image success")
	return nil
}

func installApplication(appImageName string, cmds []string, appNames []string, extension v12.ImageExtension,
	infraDriver infradriver.InfraDriver, imageEngine imageengine.Interface, mode string) error {
	if len(cmds) != 0 && len(appNames) != 0 {
		return fmt.Errorf("only one can be selected to do overwrite for launchCmds(%s) and appNames（%s）", cmds, appNames)
	}

	var launchCmds []string
	if len(cmds) != 0 {
		launchCmds = cmds
	} else {
		launchCmds = clusterruntime.GetAppLaunchCmdsByNames(appNames, extension.Applications)
	}

	clusterHosts := infraDriver.GetHostIPList()

	clusterHostsPlatform, err := infraDriver.GetHostsPlatform(clusterHosts)
	if err != nil {
		return err
	}

	imageMounter, err := imagedistributor.NewImageMounter(imageEngine, clusterHostsPlatform)
	if err != nil {
		return err
	}

	imageMountInfo, err := imageMounter.Mount(appImageName)
	if err != nil {
		return err
	}

	defer func() {
		err = imageMounter.Umount(appImageName, imageMountInfo)
		if err != nil {
			logrus.Errorf("failed to umount cluster image: %v", err)
		}
	}()

	distributor, err := imagedistributor.NewScpDistributor(imageMountInfo, infraDriver, nil)
	if err != nil {
		return err
	}

	if mode == common.ApplyModeLoadImage {
		return loadToRegistry(infraDriver, distributor)
	}

	installer := clusterruntime.NewAppInstaller(infraDriver, distributor, extension)
	err = installer.Install(infraDriver.GetHostIPListByRole(common.MASTER)[0], launchCmds)
	if err != nil {
		return err
	}

	return nil
}
