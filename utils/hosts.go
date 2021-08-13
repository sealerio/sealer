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
	"strings"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

func GetDiffHosts(hostsOld, hostsNew v1.Hosts) (add, sub []string) {
	diffMap := make(map[string]bool)
	for _, v := range hostsOld.IPList {
		diffMap[v] = true
	}
	for _, v := range hostsNew.IPList {
		if !diffMap[v] {
			add = append(add, v)
		} else {
			diffMap[v] = false
		}
	}
	for _, v := range hostsOld.IPList {
		if diffMap[v] {
			sub = append(sub, v)
		}
	}

	return
}

func IsInContainer() bool {
	data, err := ReadAll("/proc/1/environ")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "container=docker")
}

func HasSameHosts(hostsOld, hostsNew v1.Hosts) bool {
	return HasSameIPList(hostsOld.IPList, hostsNew.IPList)
}

func HasSameIPList(IPListOld, IPListNew []string) bool {
	diffMap := make(map[string]bool)
	for _, v := range IPListOld {
		diffMap[v] = true
	}
	for _, v := range IPListNew {
		if diffMap[v] {
			return true
		}
	}
	return false
}
