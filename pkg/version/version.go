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

package version

import (
	"fmt"
	"strings"
)

// Version is a string that we used to normalize version string.
type Version string

// splitVersion takes version string, and encapsulates it in comparable []string.
func splitVersion(version string) []string {
	version = strings.Replace(version, "v", "", -1)
	version = strings.Split(version, "-")[0]
	return strings.Split(version, ".")
}

// GreaterThan if givenVersion >= oldVersion return true, else return false
func (v Version) GreaterThan(oldVersion Version) (bool, error) {
	givenVersion := splitVersion(string(v))
	ov := splitVersion(string(oldVersion))

	if len(givenVersion) != 3 || len(ov) != 3 {
		return false, fmt.Errorf("error version format %s %s", v, ov)
	}
	//TODO check if necessary need v = version logic!
	if givenVersion[0] > ov[0] {
		return true, nil
	} else if givenVersion[0] < ov[0] {
		return false, nil
	}
	if givenVersion[1] > ov[1] {
		return true, nil
	} else if givenVersion[1] < ov[1] {
		return false, nil
	}
	if givenVersion[2] > ov[2] {
		return true, nil
	}
	return true, nil
}
