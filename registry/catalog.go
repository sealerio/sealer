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
	"net/url"

	"github.com/peterhellberg/link"
)

type catalogResponse struct {
	Repositories []string `json:"repositories"`
}

// Catalog returns the repositories in a registry.
func (registry *Registry) Catalog(ctx context.Context, u string) ([]string, error) {
	if u == "" {
		u = "/v2/_catalog"
	}
	uri := registry.url(u)
	registry.Logf("registry.catalog url=%s", uri)

	var response catalogResponse
	resp, err := registry.getJSON(ctx, uri, &response)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	for _, l := range link.ParseHeader(resp.Header) {
		if l.Rel == "next" {
			unescaped, _ := url.QueryUnescape(l.URI)
			repos, err := registry.Catalog(ctx, unescaped)
			if err != nil {
				return nil, err
			}
			response.Repositories = append(response.Repositories, repos...)
		}
	}

	return response.Repositories, nil
}
