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
)

type Named struct {
	domain  string // like ***.com, won't be empty
	raw     string // this name is going to be local tag name
	repo    string // k8s, sealer/k8s
	repoTag string // sealer/k8s:v1.6
	tag     string // v1.6
}

// ParseToNamed build a ImageNamed
func ParseToNamed(name string) (Named, error) {
	name = strings.TrimSpace(name)
	if err := validate(name); err != nil {
		return Named{}, err
	}

	var named Named
	named.raw = buildRaw(name)
	named.domain, named.repoTag = normalizeDomainRepoTag(name)
	named.repo, named.tag = buildRepoAndTag(named.repoTag)
	if strings.ToLower(named.repo) != named.repo {
		return named, errors.New("uppercase is not allowed in image name")
	}
	return named, nil
}

func (n Named) String() string {
	return n.Name()
}

func (n Named) Name() string {
	if n.domain == "" {
		return n.Repo()
	}
	return n.domain + "/" + n.Repo()
}

func (n Named) Domain() string {
	return n.domain
}

func (n Named) RepoTag() string {
	return n.repoTag
}

func (n Named) Raw() string {
	return n.raw
}

func (n Named) Repo() string {
	return n.repo
}

func (n Named) Tag() string {
	return n.tag
}

func (n Named) CompleteName() string {
	return n.domain + "/" + n.repoTag
}
