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

package distributionutil

import (
	"encoding/json"
	"fmt"

	"github.com/sealerio/sealer/logger"
	v1 "github.com/sealerio/sealer/types/api/v1"
	platUtil "github.com/sealerio/sealer/utils/platform"

	"github.com/opencontainers/go-digest"
)

// ManifestList this package unmarshal manifests from json into a ManifestList struct
//then choose corresponding manifest by platform
type ManifestList struct {
	List      []ImageManifest `json:"manifests"`
	MediaType string          `json:"mediaType"`
	Schema    int             `json:"schemaVersion"`
}

type ImageManifest struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	Platform  v1.Platform
	Size      int
}

func GetImageManifestDigest(payload []byte, plat v1.Platform) (digest.Digest, error) {
	var (
		manifestList ManifestList
	)

	err := json.Unmarshal(payload, &manifestList)
	if err != nil {
		return "", fmt.Errorf("json unmarshal error: %v", err)
	}

	var resDigest []digest.Digest
	for _, item := range manifestList.List {
		if platUtil.Matched(item.Platform, plat) {
			resDigest = append(resDigest, digest.Digest(item.Digest))
		}
	}

	if len(resDigest) == 0 {
		return "", fmt.Errorf("no manifest of the corresponding platform")
	}

	if len(resDigest) > 1 {
		logger.Warn("multiple matches in manifest list")
	}
	return resDigest[0], nil
}
