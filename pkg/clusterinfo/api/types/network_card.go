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

type NetWorkCard struct {
	// Name
	Name string `json:"name" yaml:"name"`

	// IP
	IP string `json:"ip" yaml:"ip"`

	// MAC
	MAC string `json:"mac" yaml:"mac"`
}

func (s NetWorkCard) Value() (driver.Value, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return driver.Value(string(result)), nil
}

func (s *NetWorkCard) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, &s)
	case string:
		return json.Unmarshal([]byte(v), &s)
	default:
		return fmt.Errorf("cannot sql.Scanner.Scan() NetWorkCard from: %#v", v)
	}
}

type NetWorkCardSlice []*NetWorkCard

func (s NetWorkCardSlice) Value() (driver.Value, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return driver.Value(string(result)), nil
}

func (s *NetWorkCardSlice) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, &s)
	case string:
		return json.Unmarshal([]byte(v), &s)
	default:
		return fmt.Errorf("cannot sql.Scanner.Scan() NetWorkCardSlice from: %#v", v)
	}
}
