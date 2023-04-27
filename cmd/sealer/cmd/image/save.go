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
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/imageengine/buildah"
)

var saveOpts *options.SaveOptions

var longNewSaveCmdDescription = `sealer save -o [output file name] [image name]`

var exampleForSaveCmd = `
save docker.io/sealerio/kubernetes:v1-22-15-sealerio-2 image to kubernetes.tar file:

  sealer save -o kubernetes.tar docker.io/sealerio/kubernetes:v1-22-15-sealerio-2`

// NewSaveCmd saveCmd represents the save command
func NewSaveCmd() *cobra.Command {
	saveCmd := &cobra.Command{
		Use:     "save",
		Short:   "save sealer image to a tar file",
		Long:    longNewSaveCmdDescription,
		Example: exampleForSaveCmd,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			saveOpts.ImageNameOrID = args[0]

			err = engine.Save(saveOpts)
			if err == nil {
				logrus.Infof("successfully save %s to %s", args[0], saveOpts.Output)
			}
			return err
		},
	}
	saveOpts = &options.SaveOptions{}
	flags := saveCmd.Flags()
	flags.StringVar(&saveOpts.Format, "format", buildah.OCIArchive, "Save image to oci-archive, oci-dir (directory with oci manifest type), docker-archive, docker-dir (directory with v2s2 manifest type)")
	flags.StringVarP(&saveOpts.Output, "output", "o", "", "Write image to a specified file")
	flags.BoolVarP(&saveOpts.Quiet, "quiet", "q", false, "Suppress the output")
	flags.StringVar(&saveOpts.TmpDir, "tmp-dir", "", "set temporary directory when save image. if not set, use system`s temporary directory")
	flags.BoolVar(&saveOpts.Compress, "compress", false, "Compress tarball image layers when saving to a directory using the 'dir' transport. (default is same compression type as source)")
	if err := saveCmd.MarkFlagRequired("output"); err != nil {
		logrus.Errorf("failed to init flag: %v", err)
		os.Exit(1)
	}

	return saveCmd
}
