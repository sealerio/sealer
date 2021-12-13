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

//this package unmarshal blobs from json into a BlobList struct
//then return a slice of blob digest
type BlobList struct {
	Layers    []Blob `json:"layers"`
	Config    Blob   `json:"config"`
	MediaType string `json:"mediaType"`
	Schema    int    `json:"schemaVersion"`
}

type Blob struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	Size      int
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
	blobDigests = append(blobDigests, digest.Digest(blobList.Config.Digest))
	for _, layer := range blobList.Layers {
		blobDigests = append(blobDigests, digest.Digest(layer.Digest))
	}
	return blobDigests, nil
}
