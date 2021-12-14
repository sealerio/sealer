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

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

//this package unmarshal manifests from json into a ManifestList struct
//then choose corresponding manifest by platform
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

func getImageManifestDigest(payload []byte, platform v1.Platform) (digest.Digest, error) {
	var manifestList ManifestList
	err := json.Unmarshal(payload, &manifestList)
	if err != nil {
		return "", fmt.Errorf("json unmarshal error: %v", err)
	}
	// look up manifest of the corresponding architecture
	for _, item := range manifestList.List {
		if equalPlatform(item.Platform, platform) {
			return digest.Digest(item.Digest), nil
		}
	}
	return "", fmt.Errorf("no manifest of the corresponding platform")
}

func equalPlatform(src, target v1.Platform) bool {
	if target.OS != "" && src.OS != target.OS {
		return false
	}

	if target.Architecture != "" && src.Architecture != target.Architecture {
		return false
	}

	if target.Variant != "" && src.Variant != target.Variant {
		return false
	}
	return true
}
