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
)

var (
	loadOpts *options.LoadOptions

	longNewLoadCmdDescription = `
  Load a sealer image from a tar archive
  Save an image to docker-archive or oci-archive on the local machine. Default is docker-archive.`

	exampleForLoadCmd = `
  sealer load -i kubernetes.tar

Specifies the temporary load directory:
  sealer load -i my.tar --tmp-dir /root/my-tmp`
)

// NewLoadCmd loadCmd represents the load command
func NewLoadCmd() *cobra.Command {
	loadCmd := &cobra.Command{
		Use:     "load",
		Short:   "load a sealer image from a tar file",
		Long:    longNewLoadCmdDescription,
		Example: exampleForLoadCmd,
		Args:    cobra.NoArgs,
		RunE:    runLoadCmd,
	}
	loadOpts = &options.LoadOptions{}
	flags := loadCmd.Flags()
	flags.StringVarP(&loadOpts.Input, "input", "i", "", "Read from specified archive file")
	flags.BoolVarP(&loadOpts.Quiet, "quiet", "q", false, "Suppress the output")
	flags.StringVar(&loadOpts.TmpDir, "tmp-dir", "", "Set temporary directory when load image. use system temporary directory is not (/var/tmp/)at present ")

	if err := loadCmd.MarkFlagRequired("input"); err != nil {
		logrus.Errorf("failed to init flag: %v", err)
		os.Exit(1)
	}
	return loadCmd
}

func runLoadCmd(cmd *cobra.Command, args []string) error {
	engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
	if err != nil {
		return err
	}

	err = engine.Load(loadOpts)
	if err == nil {
		logrus.Infof("successfully load %s to image storage", loadOpts.Input)
	}

	return err
}
