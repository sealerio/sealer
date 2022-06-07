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
	"fmt"

	"github.com/sealerio/sealer/pkg/clusterinfo"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

type HardwareResource struct {
	CPUMinimum        int32
	MemMinimum        int32
	SystemDiskMinimum int32
}

var hardwareResourceRequired = HardwareResource{
	CPUMinimum:        2,
	MemMinimum:        4,
	SystemDiskMinimum: 40,
}

type ResourceChecker struct {
}

func NewResourceChecker() Interface {
	return &ResourceChecker{}
}

func (o ResourceChecker) Check(cluster *v2.Cluster, phase string) error {
	detailed, err := clusterinfo.GetClusterInfo(cluster)
	if err != nil {
		return err
	}

	for _, instance := range detailed.InstanceInfos {
		if instance.CPU < hardwareResourceRequired.CPUMinimum {
			return fmt.Errorf("cpu cores should >=%d", hardwareResourceRequired.CPUMinimum)
		}

		if instance.Memory < hardwareResourceRequired.MemMinimum {
			return fmt.Errorf("memory capacity should >=%dGB", hardwareResourceRequired.MemMinimum)
		}

		if len(instance.SystemDisk) < 1 {
			return fmt.Errorf("systemDisk not found")
		}
		if instance.SystemDisk[0].Capacity < hardwareResourceRequired.SystemDiskMinimum {
			return fmt.Errorf("systemDisk capacity should >=%dGB", hardwareResourceRequired.SystemDiskMinimum)
		}
	}

	return nil
}
