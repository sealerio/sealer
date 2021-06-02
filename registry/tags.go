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

import "context"

type tagsResponse struct {
	Tags []string `json:"tags"`
}

// Tags returns the tags for a specific repository.
func (registry *Registry) Tags(ctx context.Context, repository string) ([]string, error) {
	url := registry.url("/v2/%s/tags/list", repository)
	registry.Logf("registry.tags url=%s repository=%s", url, repository)

	var response tagsResponse
	if _, err := registry.getJSON(ctx, url, &response); err != nil {
		return nil, err
	}

	return response.Tags, nil
}
