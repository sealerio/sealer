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

package alpha

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var clusterFile string

func NewHostAliasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host-alias",
		Short: "set host-alias for hosts via specified Clusterfile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				cf              clusterfile.Interface
				clusterFileData []byte
				err             error
			)
			logrus.Warn("sealer apply command will be deprecated in the future, please use sealer run instead.")

			if clusterFile == "" {
				return fmt.Errorf("you must input Clusterfile")
			}

			clusterFileData, err = ioutil.ReadFile(filepath.Clean(clusterFile))
			if err != nil {
				return err
			}

			cf, err = clusterfile.NewClusterFile(clusterFileData)
			if err != nil {
				return err
			}

			desiredCluster := cf.GetCluster()
			infraDriver, err := infradriver.NewInfraDriver(&desiredCluster)
			if err != nil {
				return err
			}

			// set HostAlias
			if err := infraDriver.SetClusterHostAliases(infraDriver.GetHostIPList()); err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&clusterFile, "Clusterfile", "f", "", "Clusterfile path to apply a Kubernetes cluster")
	return cmd
}
