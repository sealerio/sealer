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

package utils

import (
	"strconv"

	"github.com/sealerio/sealer/apply"
	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

func ConstructClusterFromArg(imageName string, runArgs *apply.Args) (*v2.Cluster, error) {
	resultHosts, err := getHosts(runArgs.Masters, runArgs.Nodes)
	if err != nil {
		return nil, err
	}
	cluster := v2.Cluster{
		Spec: v2.ClusterSpec{
			SSH: v1.SSH{
				User:     runArgs.User,
				Passwd:   runArgs.Password,
				PkPasswd: runArgs.PkPassword,
				Pk:       runArgs.Pk,
				Port:     strconv.Itoa(int(runArgs.Port)),
			},
			Image:   imageName,
			Hosts:   resultHosts,
			Env:     runArgs.CustomEnv,
			CMDArgs: runArgs.CMDArgs,
		},
	}
	cluster.APIVersion = common.APIVersion
	cluster.Kind = common.Kind
	cluster.Name = runArgs.ClusterName
	return &cluster, nil
}
