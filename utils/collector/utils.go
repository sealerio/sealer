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

package collector

import (
	"regexp"
	"strings"
)

var (
	validPrefixes = map[string][]string{
		"url": {"http://", "https://"},
		"git": {"git://", "git@"},
	}
	urlPathWithFragmentSuffix = regexp.MustCompile(".git(?:#.+)?$")
)

// IsURL returns true if the provided str is an HTTP(S) URL.
func IsURL(str string) bool {
	return checkURL(str, "url")
}

// IsGitURL returns true if the provided str is a git repository URL.
func IsGitURL(str string) bool {
	if IsURL(str) && urlPathWithFragmentSuffix.MatchString(str) {
		return true
	}
	return checkURL(str, "git")
}

func checkURL(str, kind string) bool {
	for _, prefix := range validPrefixes[kind] {
		if strings.HasPrefix(str, prefix) {
			return true
		}
	}
	return false
}
