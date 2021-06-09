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
	"os"

	"github.com/alibaba/sealer/common"

	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/build"
	"github.com/alibaba/sealer/logger"
)

type BuildFlag struct {
	ImageName    string
	KubefileName string
	Context      string
	BuildType    string
}

var buildConfig *BuildFlag

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "cloud image local build command line",
	Long:  `seautil build -f Kubefile -t my-kubernetes:1.18.3 . `,
	Run: func(cmd *cobra.Command, args []string) {
		conf := &build.Config{
			BuildType: buildConfig.BuildType,
		}
		builder, err := build.NewBuilder(conf)
		if err != nil {
			logger.Error(err)
			os.Exit(-1)
		}

		err = builder.Build(buildConfig.ImageName, buildConfig.Context, buildConfig.KubefileName)
		if err != nil {
			logger.Error(err)
			os.Exit(-1)
		}
	},
}

func init() {
	buildConfig = &BuildFlag{}
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringVarP(&buildConfig.KubefileName, "kubefile", "f", "", "kubefile filepath")
	buildCmd.Flags().StringVarP(&buildConfig.ImageName, "imageName", "t", "", "cluster image name")
	buildCmd.Flags().StringVarP(&buildConfig.Context, "context", "c", ".", "cluster image build context file path")
	buildCmd.Flags().StringVarP(&buildConfig.BuildType, "buildType", "b", common.LocalBuild, "specific of type is local build or cloud build default is local")
}
