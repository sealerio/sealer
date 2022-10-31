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

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"
	"github.com/sealerio/sealer/common"
	clusterruntime "github.com/sealerio/sealer/pkg/cluster-runtime"
	"github.com/sealerio/sealer/pkg/clusterfile"
	v12 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var appFlags *types.APPFlags
var longNewRunAPPCmdDescription = `sealer run-app localhost/nginx:v1`
var exampleForRunAppCmd = ``

func NewRunAPPCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:     "run-app",
		Short:   "start to run an application cluster image",
		Long:    longNewRunAPPCmdDescription,
		Example: exampleForRunAppCmd,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				cf              clusterfile.Interface
				clusterFileData []byte
				err             error
			)

			//todo grab more cluster info from api server
			clusterFileData, err = ioutil.ReadFile(common.GetDefaultClusterfile())
			if err != nil {
				return err
			}

			cf, err = clusterfile.NewClusterFile(clusterFileData)
			if err != nil {
				return err
			}

			cluster := cf.GetCluster()
			infraDriver, err := infradriver.NewInfraDriver(&cluster)
			if err != nil {
				return err
			}

			imageEngine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			extension, err := imageEngine.GetSealerImageExtension(&options.GetImageAnnoOptions{ImageNameOrID: args[0]})
			if err != nil {
				return fmt.Errorf("failed to get cluster image extension: %s", err)
			}

			if extension.Type != v12.AppInstaller {
				return fmt.Errorf("exit install process, wrong cluster image type: %s", extension.Type)
			}

			return installApplication(args[0], appFlags.LaunchCmds, extension, infraDriver, imageEngine)
		},
	}

	appFlags = &types.APPFlags{}
	runCmd.Flags().StringSliceVar(&appFlags.LaunchCmds, "cmds", []string{}, "override default LaunchCmds of clusterimage")
	//runCmd.Flags().StringSliceVar(&appFlags.LaunchArgs, "args", []string{}, "override default LaunchArgs of clusterimage")
	return runCmd
}

func installApplication(appImageName string, launchCmds []string, extension v12.ImageExtension,
	infraDriver infradriver.InfraDriver, imageEngine imageengine.Interface) error {
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
		err = imageMounter.Umount(imageMountInfo)
		if err != nil {
			logrus.Errorf("failed to umount cluster image")
		}
	}()

	distributor, err := imagedistributor.NewScpDistributor(imageMountInfo, infraDriver, nil)
	if err != nil {
		return err
	}

	//todo grab this config from cluster file, that's because it belongs to cluster level information
	var registryConfig registry.RegConfig
	var config = registry.Registry{
		Domain: registry.DefaultDomain,
		Port:   registry.DefaultPort,
	}

	registryConfig.LocalRegistry = &registry.LocalRegistry{
		DataDir:    filepath.Join(infraDriver.GetClusterRootfsPath(), "registry"),
		DeployHost: infraDriver.GetHostIPListByRole(common.MASTER)[0],
		Registry:   config,
	}

	installer := clusterruntime.NewAppInstaller(infraDriver, distributor, extension, registryConfig)
	err = installer.Install(infraDriver.GetHostIPListByRole(common.MASTER)[0], launchCmds)
	if err != nil {
		return err
	}

	return nil
}
