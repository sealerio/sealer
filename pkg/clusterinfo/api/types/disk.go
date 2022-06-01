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

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Disk struct {
	// Name
	Name string `json:"name" yaml:"name"`

	// required
	// Minimum: 1
	// TODO: Deprecated
	Required int32 `json:"required,omitempty" yaml:"required,omitempty"`

	// Capacity the total storage capacity.
	Capacity int32 `json:"capacity" yaml:"capacity,omitempty"`

	// Remain the remain storage capacity.
	Remain int32 `json:"remain,omitempty" yaml:"remain,omitempty"`

	// FSType the file system type.
	FSType string `json:"fsType" yaml:"fsType"`

	// MountPoint
	MountPoint string `json:"mountPoint" yaml:"mountPoint"`

	// Type the disk type.
	Type string `json:"type" yaml:"type"`
}

func (s Disk) Value() (driver.Value, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return driver.Value(string(result)), nil
}

func (s *Disk) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, &s)
	case string:
		return json.Unmarshal([]byte(v), &s)
	default:
		return fmt.Errorf("cannot sql.Scanner.Scan() Disk from: %#v", v)
	}
}

// DiskSlice disk slice
type DiskSlice []*Disk

func (s DiskSlice) Value() (driver.Value, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return driver.Value(string(result)), nil
}

func (s *DiskSlice) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, &s)
	case string:
		return json.Unmarshal([]byte(v), &s)
	default:
		return fmt.Errorf("cannot sql.Scanner.Scan() DiskSlice from: %#v", v)
	}
}
