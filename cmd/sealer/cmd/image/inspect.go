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
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/spf13/cobra"
)

var inspectOpts *options.InspectOptions

var exampleForInspectCmd = `sealer inspect {imageName or imageID}
sealer inspect --format '{{.OCIv1.Config.Env}}' {imageName or imageID}
`

// NewInspectCmd inspectCmd represents the inspect command
func NewInspectCmd() *cobra.Command {
	inspectCmd := &cobra.Command{
		Use:     "inspect",
		Short:   "print the image information or Clusterfile",
		Example: exampleForInspectCmd,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			inspectOpts.ImageNameOrID = args[0]
			err = engine.Inspect(inspectOpts)
			if err != nil {
				return err
			}
			return nil
		},
	}
	inspectOpts = &options.InspectOptions{}
	flags := inspectCmd.Flags()
	flags.StringVarP(&inspectOpts.Format, "format", "f", "", "use `format` as a Go template to format the output")
	flags.StringVarP(&inspectOpts.InspectType, "type", "t", "image", "look at the item of the specified `type` (container or image) and name")
	return inspectCmd
}
