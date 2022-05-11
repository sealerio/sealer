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

package slice

import (
	"strings"
	"unicode"
)

type Interface interface {
	// GetIntersection get intersection element form between slice.
	GetIntersection() []string
	// GetUnion get Union element between two slice.
	GetUnion() []string
	// GetSrcSubtraction get different element in src compare to dst.
	GetSrcSubtraction() []string
	// GetDstSubtraction get different element in dst compare to src.
	GetDstSubtraction() []string
}

type Comparator struct {
	Src []string
	Dst []string
}

func (c Comparator) GetIntersection() []string {
	var result []string
	for _, elem := range c.Src {
		// elem both exist in src and dst at the same time.
		if !NotIn(elem, c.Dst) {
			result = append(result, elem)
		}
	}
	return result
}

func (c Comparator) GetUnion() []string {
	result := c.Src
	for _, elem := range c.Dst {
		// get all elem
		if NotIn(elem, c.Src) {
			result = append(result, elem)
		}
	}
	return result
}

func (c Comparator) GetSrcSubtraction() []string {
	var result []string
	for _, elem := range c.Src {
		// get src elem which not in dst
		if NotIn(elem, c.Dst) {
			result = append(result, elem)
		}
	}
	return result
}

func (c Comparator) GetDstSubtraction() []string {
	var result []string
	for _, elem := range c.Dst {
		// get dst elem which not in src
		if NotIn(elem, c.Src) {
			result = append(result, elem)
		}
	}
	return result
}

func NewComparator(src, dst []string) Interface {
	return Comparator{
		Src: src,
		Dst: dst,
	}
}

func NotIn(key string, slice []string) bool {
	for _, s := range slice {
		if key == s {
			return false
		}
	}
	return true
}

func Reverse(s []string) []string {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func ContainPartial(list []string, partial string) (result []string) {
	for i := range list {
		if strings.Contains(list[i], partial) {
			result = append(result, list[i])
		}
	}
	return
}

func RemoveDuplicate(list []string) []string {
	var result []string
	flagMap := map[string]struct{}{}
	for _, v := range list {
		if _, ok := flagMap[v]; !ok {
			flagMap[v] = struct{}{}
			result = append(result, v)
		}
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

// Merge :merge slice type as overwrite model
func Merge(ms ...[]string) []string {
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

// ConvertToMap :if env list containers Unicode punctuation character,will ignore this element.
func ConvertToMap(env []string) map[string]string {
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

func Diff(old, new []string) (add, sub []string) {
	diffMap := make(map[string]bool)
	for _, v := range old {
		diffMap[v] = true
	}
	for _, v := range new {
		if !diffMap[v] {
			add = append(add, v)
		} else {
			diffMap[v] = false
		}
	}
	for _, v := range old {
		if diffMap[v] {
			sub = append(sub, v)
		}
	}

	return
}
