package reference

import (
	"errors"
	"strings"
	"unicode"
)

const (
	defaultDomain = "registry.cn-qingdao.aliyuncs.com"
	defaultRepo = "seadent"
	defaultTag = "latest"
	localhost = "localhost"
)

func validate(name string) error {
	if name == "" {
		return errors.New("empty image name is not allowed")
	}

	for _, c := range name {
		if unicode.IsUpper(c) {
			return errors.New("uppercase is not allowed in image name")
		}


		if unicode.IsSpace(c) {
			return errors.New("space is not allowed in image name")
		}
	}

	return nil
}

func normalizeDomainRepoTag(name string) (originDomain, domain, repoTag string) {
	ind := strings.IndexRune(name, '/')
	if ind >= 0 && (strings.ContainsAny(name[0:ind], ".:") || name[0:ind] == localhost) {
		domain = name[0:ind]
		originDomain = name[0:ind]
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
func repoAndTag(repoTag string) (string, string) {
	splits := strings.Split(repoTag, ":")
	return splits[0], splits[1]
}
