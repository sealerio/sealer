package reference

import (
	"strings"
)

type Named struct {
	domain  string // like ***.com, won't be empty
	raw     string // this name is going to be local tagname
	repo    string // k8s, seadent/k8s
	repoTag string // seadent/k8s:v1.6
	tag     string // v1.6
}

// build a ImageNamed
func ParseToNamed(name string) (Named, error) {
	name = strings.TrimSpace(name)
	err := validate(name)
	if err != nil {
		return Named{}, err
	}

	var named Named
	named.raw = buildRaw(name)
	named.domain, named.repoTag = normalizeDomainRepoTag(name)
	named.repo, named.tag = buildRepoAndTag(named.repoTag)
	return named, nil
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
