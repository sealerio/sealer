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

package save

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	v1 "github.com/sealerio/sealer/types/api/v1"

	"github.com/distribution/distribution/v3"
	"github.com/opencontainers/go-digest"
	"k8s.io/apimachinery/pkg/util/json"
)

// this package contains some utils to handle docker image name
const (
	legacyDefaultDomain = "index.docker.io"
	defaultDomain       = "docker.io"
	officialRepoName    = "library"
	defaultTag          = "latest"
)

// docker image name struct
type Named struct {
	domain string //eg. docker.io
	repo   string //eg. library/ubuntu
	tag    string //eg. latest
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

func (n Named) FullName() string {
	return n.domain + "/" + n.repo + ":" + n.tag
}

func (n Named) Domain() string {
	return n.domain
}

func (n Named) Repo() string {
	return n.repo
}

func (n Named) Tag() string {
	return n.tag
}

func splitDockerDomain(name string, registry string) (domain, remainder string) {
	i := strings.IndexRune(name, '/')
	if i == -1 || (!strings.ContainsAny(name[:i], ".:") && name[:i] != "localhost" && strings.ToLower(name[:i]) == name[:i]) {
		if registry != "" {
			domain, remainder = registry, name
		} else {
			domain, remainder = defaultDomain, name
		}
	} else {
		domain, remainder = name[:i], name[i+1:]
	}

	if domain == legacyDefaultDomain {
		domain = defaultDomain
	}
	if domain == defaultDomain && !strings.ContainsRune(remainder, '/') {
		remainder = officialRepoName + "/" + remainder
	}
	return
}

func ParseNormalizedNamed(s string, registry string) (Named, error) {
	domain, remainder := splitDockerDomain(s, registry)
	var remoteName, tag string
	if tagSep := strings.IndexRune(remainder, ':'); tagSep > -1 {
		tag = remainder[tagSep+1:]
		remoteName = remainder[:tagSep]
	} else {
		tag = defaultTag
		remoteName = remainder
	}
	if strings.ToLower(remoteName) != remoteName {
		return Named{}, fmt.Errorf("invalid reference format: repository name (%s) must be lowercase", remoteName)
	}

	named := Named{
		domain: domain,
		repo:   remoteName,
		tag:    tag,
	}
	return named, nil
}

// BlobList this package unmarshal blobs from json into a BlobList struct
// then return a slice of blob digest
type BlobList struct {
	Layers    []distribution.Descriptor `json:"layers"`
	Config    distribution.Descriptor   `json:"config"`
	MediaType string                    `json:"mediaType"`
	Schema    int                       `json:"schemaVersion"`
}

func getBlobList(blobListJSON distribution.Manifest) ([]digest.Digest, error) {
	_, list, err := blobListJSON.Payload()
	if err != nil {
		return nil, fmt.Errorf("failed to get blob list: %v", err)
	}
	var blobList BlobList
	err = json.Unmarshal(list, &blobList)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal error: %v", err)
	}
	var blobDigests []digest.Digest
	blobDigests = append(blobDigests, blobList.Config.Digest)
	for _, layer := range blobList.Layers {
		blobDigests = append(blobDigests, layer.Digest)
	}
	return blobDigests, nil
}

// ManifestList this package unmarshal manifests from json into a ManifestList struct
// then choose corresponding manifest by platform
type ManifestList struct {
	List      []ImageManifest `json:"manifests"`
	MediaType string          `json:"mediaType"`
	Schema    int             `json:"schemaVersion"`
}

type ImageManifest struct {
	Digest    string      `json:"digest"`
	MediaType string      `json:"mediaType"`
	Platform  v1.Platform `json:"platform"`
	Size      int         `json:"size"`
}

func getImageManifestDigest(payload []byte, plat v1.Platform) (digest.Digest, error) {
	var (
		manifestList ManifestList
	)

	err := json.Unmarshal(payload, &manifestList)
	if err != nil {
		return "", fmt.Errorf("json unmarshal error: %v", err)
	}

	var resDigest []digest.Digest
	for _, item := range manifestList.List {
		if matched(item.Platform, plat) {
			resDigest = append(resDigest, digest.Digest(item.Digest))
		}
	}

	if len(resDigest) == 0 {
		return "", fmt.Errorf("no manifest of the corresponding platform")
	}

	if len(resDigest) > 1 {
		logrus.Warn("multiple matches in manifest list")
	}
	return resDigest[0], nil
}

// Matched check if src == dest
func matched(src, dest v1.Platform) bool {
	if src.OS == dest.OS &&
		src.Architecture == "arm64" && dest.Architecture == "arm64" {
		return true
	}

	return src.OS == dest.OS &&
		src.Architecture == dest.Architecture &&
		src.Variant == dest.Variant
}
