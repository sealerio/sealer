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
	"strconv"

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
	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var appFlags *types.APPFlags
var exampleForRunAppCmd = ``

func NewRunAPPCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:     "run-app",
		Short:   "start to run an application cluster image",
		Long:    `sealer run-app localhost/nginx:v1`,
		Example: exampleForRunAppCmd,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return installApplication(args[0], appFlags.LaunchCmds, nil, nil)
		},
	}

	appFlags = &types.APPFlags{}
	runCmd.Flags().StringSliceVar(&appFlags.LaunchCmds, "cmds", []string{}, "override default LaunchCmds of clusterimage")
	//runCmd.Flags().StringSliceVar(&appFlags.LaunchArgs, "args", []string{}, "override default LaunchArgs of clusterimage")
	return runCmd
}

const NotAppImageError = "IsNotAppImage"

func installApplication(appImageName string, launchCmds []string, configs []v1.Config, envs []string) error {
	clusterFileData, err := ioutil.ReadFile(common.GetDefaultClusterfile())
	if err != nil {
		return err
	}

	cf, err := clusterfile.NewClusterFile(clusterFileData)
	if err != nil {
		return err
	}

	cluster := cf.GetCluster()
	infraDriver, err := infradriver.NewInfraDriver(&cluster)
	if err != nil {
		return err
	}

	// add env from app-file
	infraDriver.AddClusterEnv(envs)

	imageEngine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
	if err != nil {
		return err
	}

	// TODO get arch info from k8s
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

	extension, err := imageEngine.GetSealerImageExtension(&options.GetImageAnnoOptions{ImageNameOrID: appImageName})
	if err != nil {
		return fmt.Errorf("failed to get cluster image extension: %s", err)
	}

	if extension.Type != v12.AppInstaller {
		return fmt.Errorf(NotAppImageError)
	}

	distributor, err := imagedistributor.NewScpDistributor(imageMountInfo, infraDriver, configs)
	if err != nil {
		return err
	}

	//todo grab this config from cluster file, that's because it belongs to cluster level information
	port, err := strconv.Atoi(common.DefaultRegistryPort)
	if err != nil {
		return err
	}
	var registryConfig registry.RegConfig
	clusterENV := infraDriver.GetClusterEnv()
	var config = registry.Registry{
		Domain: clusterENV[common.EnvRegistryDomain].(string),
		Port:   port,
		Auth:   &registry.Auth{},
	}

	if userName := clusterENV[common.EnvRegistryUsername]; userName != nil {
		config.Auth.Username = userName.(string)
	}
	if password := clusterENV[common.EnvRegistryPassword]; password != nil {
		config.Auth.Password = password.(string)
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
