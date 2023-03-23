// Copyright © 2022 Alibaba Group Holding Ltd.
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

package alpha

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/common"
)

type ParserArg struct {
	Name       string
	Passwd     string
	Image      string
	Port       uint16
	Pk         string
	PkPassword string
}

var flag *ParserArg

var longGenCmdDescription = `Sealer will call kubernetes API to get masters and nodes IP info, then generate a Clusterfile. and also pull a sealer image which matches the kubernetes version.

Then you can use any sealer command to manage the cluster like:

> Scale
  sealer join --node 192.168.0.1`

var exampleForGenCmd = `The following command will generate Clusterfile used by sealer under user home dir:

  sealer alpha gen --passwd 'Sealer123' --image docker.io/sealerio/kubernetes:v1-22-15-sealerio-2
`

// NewGenCmd returns the sealer gen Cobra command
func NewGenCmd() *cobra.Command {
	genCmd := &cobra.Command{
		Use:     "gen",
		Short:   "Generate a Clusterfile to take over a normal cluster which was not deployed by sealer",
		Long:    longGenCmdDescription,
		Example: exampleForGenCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flag.Passwd == "" || flag.Image == "" {
				return fmt.Errorf("password and image name cannot be empty")
			}
			return errors.New("gen is not implemented yet")
		},
	}

	flag = &ParserArg{}
	genCmd.Flags().Uint16Var(&flag.Port, "port", 22, "set the sshd service port number for the server (default port: 22)")
	genCmd.Flags().StringVar(&flag.Pk, "pk", common.GetHomeDir()+"/.ssh/id_rsa", "set server private key")
	genCmd.Flags().StringVar(&flag.PkPassword, "pk-passwd", "", "set server private key password")
	genCmd.Flags().StringVar(&flag.Image, "image", "", "Set taken over sealer image")
	genCmd.Flags().StringVar(&flag.Name, "name", "default", "Set taken over cluster name")
	genCmd.Flags().StringVar(&flag.Passwd, "passwd", "", "Set taken over ssh passwd")

	return genCmd
}
