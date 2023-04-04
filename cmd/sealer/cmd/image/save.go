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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/imageengine/buildah"
)

var (
	saveOpts *options.SaveOptions

	longNewSaveCmdDescription = `
  sealer save -o [output file name] [image name]
  Save an image to docker-archive or oci-archive on the local machine. Default is docker-archive.`

	exampleForSaveCmd = `
  sealer save docker.io/sealerio/kubernetes:v1-22-15-sealerio-2 

Image save to kubernetes.tar file, and specify the temporary load directory:
  sealer save docker.io/sealerio/kubernetes:v1.22.15  -o kubernetes.tar --tmp-dir /root/tmp`
)

var (
	containerConfig = buildah.NewBuildahConfig()
)

// NewSaveCmd saveCmd represents the save command
func NewSaveCmd() *cobra.Command {
	saveCmd := &cobra.Command{
		Use:     "save",
		Short:   "save sealer image to a tar file",
		Long:    longNewSaveCmdDescription,
		Example: exampleForSaveCmd,
		Args:    cobra.MinimumNArgs(1),
		RunE:    runSaveCmd,
	}
	saveOpts = &options.SaveOptions{}
	flags := saveCmd.Flags()

	formatFlagName := "format"
	flags.StringVar(&saveOpts.Format, formatFlagName, buildah.OCIArchive, "Save image to oci-archive, oci-dir (directory with oci manifest type), docker-archive, docker-dir (directory with v2s2 manifest type)")

	outputFlagName := "output"
	flags.StringVarP(&saveOpts.Output, outputFlagName, "o", "", "Write to a specified file (default: stdout, which must be redirected)")

	// TODO: Waiting for implementation, not yet supported
	flags.StringVar(&loadOpts.TmpDir, "tmp-dir", "", "Set temporary directory when load image. use system temporary directory is not (/var/tmp/)at present")

	flags.BoolVarP(&saveOpts.Quiet, "quiet", "q", false, "Suppress the output")

	compressFlagName := "compress"
	flags.BoolVar(&saveOpts.Compress, compressFlagName, false, "Compress tarball image layers when saving to a directory using the 'dir' transport. (default is same compression type as source)")

	MultiImageArchiveFlagName := "multi-image-archive"
	flags.BoolVarP(&saveOpts.MultiImageArchive, MultiImageArchiveFlagName, "m", containerConfig.ContainersConfDefaultsRO.Engine.MultiImageArchive, "Interpret additional arguments as images not tags and create a multi-image-archive (only for docker-archive)")

	if err := saveCmd.MarkFlagRequired("output"); err != nil {
		logrus.WithError(err).Fatal("failed to mark flag as required")
	}

	return saveCmd
}

func runSaveCmd(cmd *cobra.Command, args []string) error {
	engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
	if err != nil {
		return err
	}

	saveOpts.ImageNameOrID = args[0]

	err = engine.Save(saveOpts)
	if err == nil {
		logrus.Infof("successfully saved %s to %s", args[0], saveOpts.Output)
	}
	return err
}
