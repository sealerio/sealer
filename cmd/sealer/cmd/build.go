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
	"github.com/containers/buildah/pkg/parse"
	"github.com/sealerio/sealer/pkg/image_adaptor"
	bc "github.com/sealerio/sealer/pkg/image_adaptor/common"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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

var buildFlags bc.BuildFlags = bc.BuildFlags{}

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build [flags] PATH",
	Short: "build a ClusterImage from a Kubefile",
	Long: `build command is used to generate a ClusterImage from specified Kubefile.
It organizes the specified Kubefile and input building context, and builds
a brand new ClusterImage.`,
	Args: cobra.MaximumNArgs(1),
	Example: `the current path is the context path, default build type is lite and use image_adaptor cache

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
		builder, err := image_adaptor.NewAdaptor()
		if err != nil {
			logrus.Fatalf("failed to initiate a builder, %v", err)
		}

		err = builder.Build(&buildFlags, args)
		if err != nil {
			logrus.Error(err)
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&buildFlags.BuildType, "mode", "m", "lite", "ClusterImage build type, default is lite")
	buildCmd.Flags().StringVarP(&buildFlags.Kubefile, "file", "f", "Kubefile", "Kubefile filepath")
	buildCmd.Flags().BoolVar(&buildFlags.NoCache, "no-cache", false, "do not use existing cached images for building. Build from the start with a new set of cached layers.")
	buildCmd.Flags().BoolVar(&buildFlags.Base, "base", true, "build with base image, default value is true.")
	buildCmd.Flags().StringSliceVarP(&buildFlags.Tags, "tag", "t", []string{}, "specify a name for ClusterImage")
	buildCmd.Flags().StringSliceVar(&buildFlags.BuildArgs, "build-arg", []string{}, "set custom image_adaptor args")
	buildCmd.Flags().StringVar(&buildFlags.Platform, "platform", parse.DefaultPlatform(), "set the target platform, like linux/amd64 or linux/amd64/v7")

	requiredFlags := []string{"tag"}
	for _, flag := range requiredFlags {
		if err := buildCmd.MarkFlagRequired(flag); err != nil {
			logrus.Fatal(err)
		}
	}
}
