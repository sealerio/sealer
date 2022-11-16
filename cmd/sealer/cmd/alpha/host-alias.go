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
	"io/ioutil"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/pkg/infradriver"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var hostAlias v2.HostAlias

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

			workClusterfile := common.GetDefaultClusterfile()
			clusterFileData, err = ioutil.ReadFile(workClusterfile)
			if err != nil {
				return err
			}

			cf, err = clusterfile.NewClusterFile(clusterFileData)
			if err != nil {
				return err
			}

			desiredCluster := cf.GetCluster()
			desiredCluster.Spec.HostAliases = append(desiredCluster.Spec.HostAliases, hostAlias)
			infraDriver, err := infradriver.NewInfraDriver(&desiredCluster)
			if err != nil {
				return err
			}

			// set HostAlias
			if err := infraDriver.SetClusterHostAliases(infraDriver.GetHostIPList()); err != nil {
				return err
			}

			return cf.SaveAll()
		},
	}
	cmd.Flags().StringVar(&hostAlias.IP, "ip", "", "host-alias ip")
	cmd.Flags().StringSliceVar(&hostAlias.Hostnames, "hostnames", []string{}, "host-alias hostnames")
	if err := cmd.MarkFlagRequired("ip"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("hostnames"); err != nil {
		panic(err)
	}
	return cmd
}
