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

package registry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/opencontainers/go-digest"
)

// Delete removes a repository digest from the registry.
// https://docs.docker.com/registry/spec/api/#deleting-an-image
func (registry *Registry) Delete(ctx context.Context, repository string, digest digest.Digest) (err error) {
	url := registry.url("/v2/%s/manifests/%s", repository, digest)
	registry.Logf("registry.manifests.delete url=%s repository=%s digest=%s",
		url, repository, digest)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Accept", fmt.Sprintf("%s;q=0.9", schema2.MediaTypeManifest))
	resp, err := registry.Client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusNotFound {
		return nil
	}

	return fmt.Errorf("got status code: %d", resp.StatusCode)
}
