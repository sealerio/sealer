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

package reference

import (
	"errors"
	"strings"
	"unicode"
)

const (
	defaultDomain = "docker.io"
	defaultRepo   = "sealerio"
	defaultTag    = "latest"
	localhost     = "localhost"
)

func validate(name string) error {
	if name == "" {
		return errors.New("empty image name is not allowed")
	}

	for _, c := range name {
		if unicode.IsSpace(c) {
			return errors.New("space is not allowed in image name")
		}
	}

	return nil
}

func normalizeDomainRepoTag(name string) (domain, repoTag string) {
	ind := strings.IndexRune(name, '/')
	if ind >= 0 && (strings.ContainsAny(name[0:ind], ".:") || name[0:ind] == localhost) {
		domain = name[0:ind]
		repoTag = name[ind+1:]
	} else {
		domain = defaultDomain
		repoTag = name
	}
	if domain == defaultDomain && !strings.ContainsRune(repoTag, '/') {
		repoTag = defaultRepo + "/" + repoTag
	}
	if !strings.ContainsRune(repoTag, ':') {
		repoTag = repoTag + ":" + defaultTag
	}
	return
}

// input: urlImageName could be like "***.com/k8s:v1.1" or "k8s:v1.1"
// output: like "k8s:v1.1"
func buildRepoAndTag(repoTag string) (string, string) {
	splits := strings.Split(repoTag, ":")
	return splits[0], splits[1]
}

func buildRaw(name string) string {
	i := strings.LastIndexByte(name, ':')
	if i == -1 {
		return name + ":" + defaultTag
	}
	if i > strings.LastIndexByte(name, '/') {
		return name
	}
	return name + ":" + defaultTag
}
