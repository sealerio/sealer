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

	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/build"
	"github.com/alibaba/sealer/logger"
)

type BuildFlag struct {
	ImageName    string
	KubefileName string
	BuildType    string
	NoCache      bool
}

var buildConfig *BuildFlag

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build [flags] PATH",
	Short: "cloud image local build command line",
	Long:  "sealer build -f Kubefile -t my-kubernetes:1.19.9 [--mode cloud|container|lite] [--no-cache]",
	Args:  cobra.ExactArgs(1),
	Example: `the current path is the context path, default build type is cloud and use build cache

cloud build:
	sealer build -f Kubefile -t my-kubernetes:1.19.9

container build:
	sealer build -f Kubefile -t my-kubernetes:1.19.9 -m container

lite build:
	sealer build -f Kubefile -t my-kubernetes:1.19.9 --mode lite

build without cache:
	sealer build -f Kubefile -t my-kubernetes:1.19.9 --no-cache
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		conf := &build.Config{
			BuildType: buildConfig.BuildType,
			NoCache:   buildConfig.NoCache,
			ImageName: buildConfig.ImageName,
		}

		builder, err := build.NewBuilder(conf)
		if err != nil {
			return err
		}

		return builder.Build(buildConfig.ImageName, args[0], buildConfig.KubefileName)
	},
}

func init() {
	buildConfig = &BuildFlag{}
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringVarP(&buildConfig.BuildType, "buildType", "b", "", "cluster image build type, default is cloud")
	buildCmd.Flags().StringVarP(&buildConfig.KubefileName, "kubefile", "f", "Kubefile", "kubefile filepath")
	buildCmd.Flags().StringVarP(&buildConfig.ImageName, "imageName", "t", "", "cluster image name")
	buildCmd.Flags().BoolVar(&buildConfig.NoCache, "no-cache", false, "build without cache")
	if err := buildCmd.MarkFlagRequired("imageName"); err != nil {
		logger.Error("failed to init flag: %v", err)
		os.Exit(1)
	}
}
