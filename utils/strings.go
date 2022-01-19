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
	"bytes"
	"net"
	"sort"
	"strings"
)

func NotIn(key string, slice []string) bool {
	for _, s := range slice {
		if key == s {
			return false
		}
	}
	return true
}

func InList(key string, slice []string) bool {
	return !NotIn(key, slice)
}

func NotInIPList(key string, slice []string) bool {
	for _, s := range slice {
		if s == "" {
			continue
		}
		if key == strings.Split(s, ":")[0] {
			return false
		}
	}
	return true
}

// ReduceStrSlice get a slice of src containing dst elements
func ReduceStrSlice(src, dst []string) []string {
	var ipList []string
	for _, ip := range src {
		if !NotIn(ip, dst) {
			ipList = append(ipList, ip)
		}
	}
	return ipList
}

// RemoveStrSlice remove dst element from src slice
func RemoveStrSlice(src, dst []string) []string {
	var ipList []string
	for _, ip := range src {
		if NotIn(ip, dst) {
			ipList = append(ipList, ip)
		}
	}
	return ipList
}

// AppendDiffSlice append elements of dst slices that do not exist in src to src slices
func AppendDiffSlice(src, dst []string) []string {
	for _, ip := range dst {
		if NotIn(ip, src) {
			src = append(src, ip)
		}
	}
	return src
}

func SortIPList(iplist []string) {
	realIPs := make([]net.IP, 0, len(iplist))
	for _, ip := range iplist {
		realIPs = append(realIPs, net.ParseIP(ip))
	}

	sort.Slice(realIPs, func(i, j int) bool {
		return bytes.Compare(realIPs[i], realIPs[j]) < 0
	})

	for i := range realIPs {
		iplist[i] = realIPs[i].String()
	}
}

func Reverse(s []string) []string {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func ContainList(list []string, toComplete string) (containerList []string) {
	for i := range list {
		if strings.Contains(list[i], toComplete) {
			containerList = append(containerList, list[i])
		}
	}
	return
}

func DedupeStrSlice(in []string) []string {
	m := make(map[string]struct{})
	var res []string
	for _, s := range in {
		if _, ok := m[s]; !ok {
			res = append(res, s)
			m[s] = struct{}{}
		}
	}
	return res
}
