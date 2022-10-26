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

	"github.com/sealerio/sealer/common"
	clusterruntime "github.com/sealerio/sealer/pkg/cluster-runtime"
	"github.com/sealerio/sealer/pkg/clusterfile"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
	osi "github.com/sealerio/sealer/utils/os"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewApplyCmd() *cobra.Command {
	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "apply a Kubernetes cluster via specified Clusterfile",
		Long: `apply command is used to apply a Kubernetes cluster via specified Clusterfile.
If the Clusterfile is applied first time, Kubernetes cluster will be created. Otherwise, sealer
will apply the diff change of current Clusterfile and the original one.`,
		Example: `sealer apply -f Clusterfile`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				cf              clusterfile.Interface
				clusterFileData []byte
				err             error
			)

			if osi.IsFileExist(common.DefaultKubeConfigFile()) {
				return fmt.Errorf("the cluster already exists")
			}

			if clusterFile == "" {
				return fmt.Errorf("you must input Clusterfile")
			}

			clusterFileData, err = ioutil.ReadFile(filepath.Clean(clusterFile))
			if err != nil {
				return err
			}

			cf, err = clusterfile.NewClusterFile(clusterFileData)
			if err != nil {
				return err
			}
			//save desired clusterfile
			if err = cf.SaveAll(); err != nil {
				return err
			}

			cluster := cf.GetCluster()
			infraDriver, err := infradriver.NewInfraDriver(&cluster)
			if err != nil {
				return err
			}

			var (
				clusterLaunchCmds = infraDriver.GetClusterLaunchCmds()
				clusterHosts      = infraDriver.GetHostIPList()
				clusterImageName  = cluster.Spec.Image
			)

			clusterHostsPlatform, err := infraDriver.GetHostsPlatform(clusterHosts)
			if err != nil {
				return err
			}

			imageEngine, err := imageengine.NewImageEngine(imagecommon.EngineGlobalConfigurations{})
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
				err = imageMounter.Umount(imageMountInfo)
				if err != nil {
					logrus.Errorf("failed to umount cluster image")
				}
			}()

			distributor, err := imagedistributor.NewScpDistributor(imageMountInfo, infraDriver, cf.GetConfigs())
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
				Distributor:       distributor,
				ImageEngine:       imageEngine,
				Plugins:           plugins,
				ClusterLaunchCmds: clusterLaunchCmds,
				ClusterImageImage: clusterImageName,
			}

			if cf.GetKubeadmConfig() != nil {
				runtimeConfig.KubeadmConfig = *cf.GetKubeadmConfig()
			}

			installer, err := clusterruntime.NewInstaller(infraDriver, *runtimeConfig)
			if err != nil {
				return err
			}

			err = installer.Install()
			if err != nil {
				return err
			}

			//save clusterfile
			if err = cf.SaveAll(); err != nil {
				return err
			}
			return nil
		},
	}
	applyCmd.Flags().StringVarP(&clusterFile, "Clusterfile", "f", "Clusterfile", "Clusterfile path to apply a Kubernetes cluster")
	applyCmd.Flags().BoolVar(&ForceDelete, "force", false, "force to delete the specified cluster if set true")
	return applyCmd
}
