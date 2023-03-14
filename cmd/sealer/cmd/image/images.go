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
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
)

var imagesOpts *options.ImagesOptions

var longNewListCmdDescription = ``

var exampleForListCmd = `
  sealer images
`

func NewListCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "images",
		Short: "list all sealer images on the local node",
		// TODO: add long description.
		Long:    longNewListCmdDescription,
		Args:    cobra.NoArgs,
		Example: exampleForListCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}
			return engine.Images(imagesOpts)
		},
	}
	imagesOpts = &options.ImagesOptions{}
	flags := listCmd.Flags()
	flags.BoolVarP(&imagesOpts.All, "all", "a", false, "show all images, including intermediate images from a build")
	flags.BoolVar(&imagesOpts.Digests, "digests", false, "show digests")
	flags.BoolVar(&imagesOpts.JSON, "json", false, "output in JSON format")
	flags.BoolVarP(&imagesOpts.NoHeading, "noheading", "n", false, "do not print column headings")
	flags.BoolVar(&imagesOpts.NoTrunc, "no-trunc", false, "do not truncate output")
	flags.BoolVarP(&imagesOpts.Quiet, "quiet", "q", false, "display only image IDs")
	flags.BoolVarP(&imagesOpts.History, "history", "", false, "display the image name history")

	return listCmd
}
