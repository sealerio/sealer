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

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"
	"github.com/sealerio/sealer/cmd/sealer/cmd/utils"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/application"
	clusterruntime "github.com/sealerio/sealer/pkg/cluster-runtime"
	"github.com/sealerio/sealer/pkg/clusterfile"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	imagev1 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/registry"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/platform"
)

var runFlags *types.RunFlags

var longNewRunCmdDescription = `sealer run docker.io/sealerio/kubernetes:v1.22.15 --masters [arg] --nodes [arg]`

var exampleForRunCmd = `
run cluster by Clusterfile: 
  sealer run -f Clusterfile
run cluster by CLI flags:
  sealer run docker.io/sealerio/kubernetes:v1.22.15 -m 172.28.80.01 -n 172.28.80.02 -p Sealer123
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

			imageEngine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			id, err := imageEngine.Pull(&options.PullOptions{
				Quiet:      false,
				PullPolicy: "missing",
				Image:      args[0],
				Platform:   "local",
			})
			if err != nil {
				return err
			}

			imageSpec, err := imageEngine.Inspect(&options.InspectOptions{ImageNameOrID: id})
			if err != nil {
				return fmt.Errorf("failed to get cluster image extension: %s", err)
			}

			if imageSpec.ImageExtension.Type == imagev1.AppInstaller {
				app := utils.ConstructApplication(nil, runFlags.Cmds, runFlags.AppNames)

				return runApplicationImage(&RunApplicationImageRequest{
					ImageName:   args[0],
					Application: app,
					Envs:        runFlags.CustomEnv,
					ImageEngine: imageEngine,
					Extension:   imageSpec.ImageExtension,
					Configs:     nil,
					RunMode:     runFlags.Mode,
				})
			}

			clusterFromFlag, err := utils.ConstructClusterForRun(args[0], runFlags)
			if err != nil {
				return err
			}

			clusterData, err := yaml.Marshal(clusterFromFlag)
			if err != nil {
				return err
			}

			cf, err := clusterfile.NewClusterFile(clusterData)
			if err != nil {
				return err
			}

			return runClusterImage(imageEngine, cf, imageSpec, runFlags.Mode)
		},
	}
	runFlags = &types.RunFlags{}
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

func runWithClusterfile(clusterFile string, runFlags *types.RunFlags) error {
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
		return fmt.Errorf("failed to get cluster image extension: %s", err)
	}

	if imageSpec.ImageExtension.Type == imagev1.AppInstaller {
		app := utils.ConstructApplication(cf.GetApplication(), cluster.Spec.CMD, cluster.Spec.APPNames)

		return runApplicationImage(&RunApplicationImageRequest{
			ImageName:   imageName,
			Application: app,
			Envs:        runFlags.CustomEnv,
			ImageEngine: imageEngine,
			Extension:   imageSpec.ImageExtension,
			Configs:     cf.GetConfigs(),
			RunMode:     runFlags.Mode,
		})
	}

	return runClusterImage(imageEngine, cf, imageSpec, runFlags.Mode)
}

func runClusterImage(imageEngine imageengine.Interface, cf clusterfile.Interface, imageSpec *imagev1.ImageSpec, mode string) error {
	cluster := cf.GetCluster()
	infraDriver, err := infradriver.NewInfraDriver(&cluster)
	if err != nil {
		return err
	}

	clusterHosts := infraDriver.GetHostIPList()
	clusterImageName := infraDriver.GetClusterImageName()

	logrus.Infof("start to create new cluster with image: %s", clusterImageName)

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
		return clusterruntime.LoadToRegistry(infraDriver, distributor)
	}

	plugins, err := loadPluginsFromImage(imageMountInfo)
	if err != nil {
		return err
	}

	if cf.GetPlugins() != nil {
		plugins = append(plugins, cf.GetPlugins()...)
	}

	runtimeConfig := &clusterruntime.RuntimeConfig{
		Distributor:            distributor,
		Plugins:                plugins,
		ContainerRuntimeConfig: cluster.Spec.ContainerRuntime,
	}

	if cf.GetKubeadmConfig() != nil {
		runtimeConfig.KubeadmConfig = *cf.GetKubeadmConfig()
	}

	installer, err := clusterruntime.NewInstaller(infraDriver, *runtimeConfig, clusterruntime.GetClusterInstallInfo(imageSpec.ImageExtension.Labels, cluster.Spec.ContainerRuntime))
	if err != nil {
		return err
	}

	//we need to save desired clusterfile to local disk temporarily
	//and will use it later to clean the cluster node if apply failed.
	if err = cf.SaveAll(clusterfile.SaveOptions{}); err != nil {
		return err
	}

	// install cluster
	err = installer.Install()
	if err != nil {
		return err
	}

	confPath := clusterruntime.GetClusterConfPath(imageSpec.ImageExtension.Labels)

	cmds := infraDriver.GetClusterLaunchCmds()
	appNames := infraDriver.GetClusterLaunchApps()

	// TODO valid construct application
	// merge to application between v2.ClusterSpec, v2.Application and image extension
	v2App, err := application.NewV2Application(utils.ConstructApplication(cf.GetApplication(), cmds, appNames), imageSpec.ImageExtension)
	if err != nil {
		return fmt.Errorf("failed to parse application from Clusterfile:%v ", err)
	}

	// install application
	if err = v2App.Launch(infraDriver); err != nil {
		return err
	}
	if err = v2App.Save(application.SaveOptions{}); err != nil {
		return err
	}

	//save and commit
	if err = cf.SaveAll(clusterfile.SaveOptions{CommitToCluster: true, ConfPath: confPath}); err != nil {
		return err
	}

	logrus.Infof("succeeded in creating new cluster with image %s", clusterImageName)

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
	regConfig := infraDriver.GetClusterRegistry()
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
	if !*regConfig.LocalRegistry.HA {
		deployHosts = []net.IP{master0}
	}

	if err := distributor.DistributeRegistry(deployHosts, filepath.Join(infraDriver.GetClusterRootfsPath(), "registry")); err != nil {
		return err
	}

	logrus.Infof("load image success")
	return nil
}

type RunApplicationImageRequest struct {
	ImageName                 string
	Application               *v2.Application
	Envs                      []string
	ImageEngine               imageengine.Interface
	Extension                 imagev1.ImageExtension
	Configs                   []v1.Config
	RunMode                   string
	IgnorePrepareAppMaterials bool
}

func runApplicationImage(request *RunApplicationImageRequest) error {
	logrus.Infof("start to install application: %s", request.ImageName)

	v2App, err := application.NewV2Application(request.Application, request.Extension)
	if err != nil {
		return fmt.Errorf("failed to parse application:%v ", err)
	}

	cf, _, err := clusterfile.GetActualClusterFile()
	if err != nil {
		return err
	}

	cluster := cf.GetCluster()
	infraDriver, err := infradriver.NewInfraDriver(&cluster)
	if err != nil {
		return err
	}
	infraDriver.AddClusterEnv(request.Envs)

	if !request.IgnorePrepareAppMaterials {
		if err := prepareMaterials(infraDriver, request.ImageEngine, v2App,
			request.ImageName, request.RunMode, request.Configs); err != nil {
			return err
		}
	}

	if err = v2App.Launch(infraDriver); err != nil {
		return err
	}
	if err = v2App.Save(application.SaveOptions{}); err != nil {
		return err
	}

	logrus.Infof("succeeded in installing new app with image %s", request.ImageName)

	return nil
}

func prepareMaterials(infraDriver infradriver.InfraDriver, imageEngine imageengine.Interface,
	v2App application.Interface,
	appImageName, mode string, configs []v1.Config) error {
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

	for _, info := range imageMountInfo {
		err = v2App.FileProcess(info.MountDir)
		if err != nil {
			return errors.Wrapf(err, "failed to execute file processor")
		}
	}

	distributor, err := imagedistributor.NewScpDistributor(imageMountInfo, infraDriver, configs)
	if err != nil {
		return err
	}

	if mode == common.ApplyModeLoadImage {
		return loadToRegistry(infraDriver, distributor)
	}

	masters := infraDriver.GetHostIPListByRole(common.MASTER)
	regConfig := infraDriver.GetClusterRegistry()
	// distribute rootfs
	if err := distributor.Distribute(masters, infraDriver.GetClusterRootfsPath()); err != nil {
		return err
	}

	//if we use local registry service, load container image to registry
	if regConfig.LocalRegistry == nil {
		return nil
	}
	deployHosts := masters
	if !*regConfig.LocalRegistry.HA {
		deployHosts = []net.IP{masters[0]}
	}

	registryConfigurator, err := registry.NewConfigurator(deployHosts,
		containerruntime.Info{},
		regConfig, infraDriver, distributor)
	if err != nil {
		return err
	}

	registryDriver, err := registryConfigurator.GetDriver()
	if err != nil {
		return err
	}

	err = registryDriver.UploadContainerImages2Registry()
	if err != nil {
		return err
	}
	return nil
}
