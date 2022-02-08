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

	"github.com/alibaba/sealer/utils"

	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/build"
	"github.com/alibaba/sealer/logger"
)

type BuildFlag struct {
	ImageName    string
	KubefileName string
	BuildType    string
	BuildArgs    []string
	NoCache      bool
	Base         bool
}

var buildConfig *BuildFlag

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build [flags] PATH",
	Short: "Build an cloud image from a Kubefile",
	Long:  "sealer build -f Kubefile -t my-kubernetes:1.19.9 [--mode cloud|container|lite] [--no-cache]",
	Args:  cobra.ExactArgs(1),
	Example: `the current path is the context path, default build type is lite and use build cache

lite build:
	sealer build -f Kubefile -t my-kubernetes:1.19.9 .

container build:
	sealer build -f Kubefile -t my-kubernetes:1.19.9 -m container .

cloud build:
	sealer build -f Kubefile -t my-kubernetes:1.19.9 --mode cloud .

build without cache:
	sealer build -f Kubefile -t my-kubernetes:1.19.9 --no-cache .

build without base:
	sealer build -f Kubefile -t my-kubernetes:1.19.9 --base=false .

build with args:
	sealer build -f Kubefile -t my-kubernetes:1.19.9 --build-arg MY_ARG=abc,PASSWORD=Sealer123 .
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		conf := &build.Config{
			BuildType: buildConfig.BuildType,
			NoCache:   buildConfig.NoCache,
			ImageName: buildConfig.ImageName,
			NoBase:    !buildConfig.Base,
			BuildArgs: utils.ConvertEnvListToMap(buildConfig.BuildArgs),
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
	buildCmd.Flags().StringVarP(&buildConfig.BuildType, "mode", "m", "lite", "cluster image build type, default is lite")
	buildCmd.Flags().StringVarP(&buildConfig.KubefileName, "kubefile", "f", "Kubefile", "kubefile filepath")
	buildCmd.Flags().StringVarP(&buildConfig.ImageName, "imageName", "t", "", "cluster image name")
	buildCmd.Flags().BoolVar(&buildConfig.NoCache, "no-cache", false, "build without cache")
	buildCmd.Flags().BoolVar(&buildConfig.Base, "base", true, "build with base image,default value is true.")
	buildCmd.Flags().StringSliceVar(&buildConfig.BuildArgs, "build-arg", []string{}, "set custom build args")

	if err := buildCmd.MarkFlagRequired("imageName"); err != nil {
		logger.Error("failed to init flag: %v", err)
		os.Exit(1)
	}
}
