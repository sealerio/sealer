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
	Platform  Platform
	Size      int
}

type Platform struct {
	Arch    string `json:"architecture"`
	Os      string `json:"os"`
	Variant string `json:"variant,omitempty"`
}

func getImageManifestDigest(manifestListJSON distribution.Manifest, arch string) (digest.Digest, error) {
	_, list, err := manifestListJSON.Payload()
	if err != nil {
		return "", fmt.Errorf("get payload error: %v", err)
	}
	var manifestList ManifestList
	err = json.Unmarshal(list, &manifestList)
	if err != nil {
		return "", fmt.Errorf("json unmarshal error: %v", err)
	}
	// look up manifest of the corresponding architecture
	for _, item := range manifestList.List {
		if item.Platform.Arch == arch {
			return digest.Digest(item.Digest), nil
		}
	}
	return "", fmt.Errorf("no manifest of the corresponding architecture")
}
