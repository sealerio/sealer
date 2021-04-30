package reference

import (
	"strings"
)

type Named struct {
	domain  string // like ***.com, won't be empty
	rawName string // almost same as it goes in, but tag would be set to latest if empty
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

	named := Named{}
	named.rawName, named.domain, named.repoTag = normalizeDomainRepoTag(name)
	named.rawName = strings.TrimPrefix(named.rawName+"/"+named.repoTag, "/")
	named.repo, named.tag = repoAndTag(named.repoTag)
	return named, nil
}

func (n Named) Domain() string {
	return n.domain
}

func (n Named) RepoTag() string {
	return n.repoTag
}

func (n Named) Raw() string {
	return n.rawName
}

func (n Named) Repo() string {
	return n.repo
}

func (n Named) Tag() string {
	return n.tag
}
