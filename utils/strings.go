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
	"unicode"
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

func ConvertMapToEnvList(m map[string]string) []string {
	result := []string{}
	for k, v := range m {
		result = append(result, k+"="+v)
	}
	return result
}

func IsLetterOrNumber(k string) bool {
	for _, r := range k {
		if r == '_' {
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}

// MergeMap :merge map type as overwrite model
func MergeMap(ms ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range ms {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}

// MergeSlice :merge slice type as overwrite model
func MergeSlice(ms ...[]string) []string {
	var base []string
	diffMap := make(map[string]bool)
	for i, s := range ms {
		if i == 0 {
			base = s
			for _, v := range base {
				diffMap[v] = true
			}
		}

		for _, v := range s {
			if !diffMap[v] {
				base = append(base, v)
				diffMap[v] = true
			}
		}
	}
	return base
}

// ConvertEnvListToMap :if env list containers Unicode punctuation character,will ignore this element.
func ConvertEnvListToMap(env []string) map[string]string {
	envs := map[string]string{}
	var k, v string
	for _, e := range env {
		if e == "" {
			continue
		}
		i := strings.Index(e, "=")
		if i < 0 {
			k = e
		} else {
			k = e[:i]
			v = e[i+1:]
		}
		// ensure map key not containers special character.
		if !IsLetterOrNumber(k) {
			continue
		}
		envs[k] = v
	}
	return envs
}

func DiffSlice(hostsOld, hostsNew []string) (add, sub []string) {
	diffMap := make(map[string]bool)
	for _, v := range hostsOld {
		diffMap[v] = true
	}
	for _, v := range hostsNew {
		if !diffMap[v] {
			add = append(add, v)
		} else {
			diffMap[v] = false
		}
	}
	for _, v := range hostsOld {
		if diffMap[v] {
			sub = append(sub, v)
		}
	}

	return
}
