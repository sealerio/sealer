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

package image

import (
	"fmt"

	"github.com/containers/buildah/pkg/parse"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
)

var pullOpts *options.PullOptions

var longNewPullCmdDescription = ``

var exampleForPullCmd = `
  sealer pull docker.io/sealerio/kubernetes:v1-22-15-sealerio-2
  sealer pull docker.io/sealerio/kubernetes:v1-22-15-sealerio-2 --platform linux/amd64
`

// NewPullCmd pullCmd represents the pull command
func NewPullCmd() *cobra.Command {
	pullCmd := &cobra.Command{
		Use:     "pull",
		Short:   "pull sealer image from a registry to local",
		Long:    longNewPullCmdDescription,
		Example: exampleForPullCmd,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}
			pullOpts.Image = args[0]
			imageID, err := engine.Pull(pullOpts)
			if err != nil {
				return fmt.Errorf("failed to pull image: %s: %v", pullOpts.Image, err)
			}

			logrus.Infof("successful pull %s with the image ID: %s", pullOpts.Image, imageID)
			return err
		},
	}
	pullOpts = &options.PullOptions{}
	pullCmd.Flags().StringVar(&pullOpts.Platform, "platform", parse.DefaultPlatform(), "prefer OS/ARCH instead of the current operating system and architecture for choosing images")
	pullCmd.Flags().StringVar(&pullOpts.PullPolicy, "policy", "always", "missing, always, ifnewer or never.")
	pullCmd.Flags().BoolVarP(&pullOpts.Quiet, "quiet", "q", false, "don't output progress information when pulling images")
	pullCmd.Flags().BoolVar(&pullOpts.SkipTLSVerify, "skip-tls-verify", false, "default is requiring HTTPS and verify certificates when accessing the registry.")
	return pullCmd
}
