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

package types

// InstanceInfo instance info
type InstanceInfo struct {
	// host name
	HostName string `json:"hostName,omitempty" yaml:"hostName,omitempty"`

	// OS
	OS string `json:"os" yaml:"os"`

	// OS Version
	OSVersion string `json:"osVersion,omitempty" yaml:"osVersion,omitempty"`

	// Arch
	Arch string `json:"arch,omitempty" yaml:"arch,omitempty"`

	// Kernel
	Kernel string `json:"kernel" yaml:"kernel"`

	// cpu
	CPU int32 `json:"cpu" yaml:"cpu" validate:"required"`

	// memory
	Memory int32 `json:"memory" yaml:"memory" validate:"required"`

	// system disk
	SystemDisk DiskSlice `json:"systemDisk,omitempty" yaml:"systemDisk,omitempty" gorm:"type:varchar(1000)" validate:"required"`

	// data disk
	DataDisk DiskSlice `json:"dataDisk,omitempty" yaml:"dataDisk,omitempty" gorm:"type:varchar(1000)"`

	// private IP
	PrivateIP string `json:"privateIP" yaml:"privateIP" validate:"required,ip"`

	// NetworkCards
	NetworkCards NetWorkCardSlice `json:"networkCards" yaml:"networkCards" gorm:"type:text"`

	// root password
	RootPassword string `json:"rootPassword" yaml:"rootPassword"`

	TimeSyncStatus TimeSyncStatus `json:"timeSyncStatus" yaml:"timeSyncStatus"`
}
