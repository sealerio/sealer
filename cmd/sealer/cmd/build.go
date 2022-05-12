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

	"github.com/sealerio/sealer/utils/strings"

	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/build"
	"github.com/sealerio/sealer/logger"
	"github.com/sealerio/sealer/utils/platform"
)

type BuildFlag struct {
	ImageName    string
	KubefileName string
	BuildType    string
	BuildArgs    []string
	Platform     string
	NoCache      bool
	Base         bool
}

var buildConfig *BuildFlag

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build [flags] PATH",
	Short: "build a CloudImage from a Kubefile",
	Long: `build command is used to build a CloudImage from specified Kubefile.
It organizes the specified Kubefile and input building context, and builds
a brand new CloudImage.`,
	Args: cobra.ExactArgs(1),
	Example: `the current path is the context path, default build type is lite and use build cache

build:
	sealer build -f Kubefile -t my-kubernetes:1.19.8 .

build without cache:
	sealer build -f Kubefile -t my-kubernetes:1.19.8 --no-cache .

build without base:
	sealer build -f Kubefile -t my-kubernetes:1.19.8 --base=false .

build with args:
	sealer build -f Kubefile -t my-kubernetes:1.19.8 --build-arg MY_ARG=abc,PASSWORD=Sealer123 .
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		buildContext := args[0]

		targetPlatforms, err := platform.GetPlatform(buildConfig.Platform)
		if err != nil {
			return err
		}
		for _, tp := range targetPlatforms {
			p := tp
			conf := &build.Config{
				BuildType: buildConfig.BuildType,
				NoCache:   buildConfig.NoCache,
				ImageName: buildConfig.ImageName,
				NoBase:    !buildConfig.Base,
				BuildArgs: strings.ConvertToMap(buildConfig.BuildArgs),
				Platform:  *p,
			}
			builder, err := build.NewBuilder(conf)
			if err != nil {
				return err
			}
			err = builder.Build(buildConfig.ImageName, buildContext, buildConfig.KubefileName)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	buildConfig = &BuildFlag{}
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringVarP(&buildConfig.BuildType, "mode", "m", "lite", "CloudImage build type, default is lite")
	buildCmd.Flags().StringVarP(&buildConfig.KubefileName, "kubefile", "f", "Kubefile", "Kubefile filepath")
	buildCmd.Flags().StringVarP(&buildConfig.ImageName, "imageName", "t", "", "the name of CloudImage")
	buildCmd.Flags().BoolVar(&buildConfig.NoCache, "no-cache", false, "build without cache")
	buildCmd.Flags().BoolVar(&buildConfig.Base, "base", true, "build with base image, default value is true.")
	buildCmd.Flags().StringSliceVar(&buildConfig.BuildArgs, "build-arg", []string{}, "set custom build args")
	buildCmd.Flags().StringVar(&buildConfig.Platform, "platform", "", "set CloudImage platform. If not set, keep same platform with runtime")

	if err := buildCmd.MarkFlagRequired("imageName"); err != nil {
		logger.Error("failed to init flag: %v", err)
		os.Exit(1)
	}
}
