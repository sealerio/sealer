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

package checker

import (
	"github.com/pkg/errors"
	"github.com/sealerio/sealer/pkg/clusterinfo"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

type HostNameChecker struct {
}

func NewHostNameChecker() Interface {
	return &HostNameChecker{}
}

func (h HostNameChecker) Check(cluster *v2.Cluster, phase string) error {
	detailed, err := clusterinfo.GetClusterInfo(cluster)
	if err != nil {
		return err
	}
	hostNameMap := make(map[string]string, len(detailed.InstanceInfos))
	for _, instance := range detailed.InstanceInfos {
		if preIP, found := hostNameMap[instance.HostName]; found {
			return errors.Errorf("hostname of %s is duplicate with host %s", instance.PrivateIP, preIP)
		}
		hostNameMap[instance.HostName] = instance.PrivateIP
	}
	return nil
}
