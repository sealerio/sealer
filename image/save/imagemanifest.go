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
	"encoding/json"
	"fmt"

	distribution "github.com/distribution/distribution/v3"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

//this package unmarshal manifests from json into a ManifestList struct
//then choose corresponding manifest by arch

type ManifestList struct {
	List      []ImageMainfest `json:"manifests"`
	MediaType string          `json:"mediaType"`
	Schema    int             `json:"schemaVersion"`
}

type ImageMainfest struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	Platform  v1.Platform
	Size      int
}

func getImageManifestDigest(manifestListJSON distribution.Manifest, platform v1.Platform) (digest.Digest, error) {
	_, list, err := manifestListJSON.Payload()
	if err != nil {
		return "", fmt.Errorf("failed to get manifestList: %v", err)
	}
	var manifestList ManifestList
	err = json.Unmarshal(list, &manifestList)
	if err != nil {
		return "", fmt.Errorf("json unmarshal error: %v", err)
	}
	// look up manifest of the corresponding architecture
	for _, item := range manifestList.List {
		if equalPlatForm(item.Platform, platform) {
			return digest.Digest(item.Digest), nil
		}
	}
	return "", fmt.Errorf("no manifest of the corresponding architecture")
}

func equalPlatForm(src, target v1.Platform) bool {
	if src.OS != "" && src.OS != target.OS {
		return false
	}

	if src.Architecture != "" && src.Architecture != target.Architecture {
		return false
	}

	if src.Variant != "" && src.Variant != target.Variant {
		return false
	}
	return true
}
