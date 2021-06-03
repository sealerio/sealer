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
	"net/http"
	"strings"
)

// Pingable checks pingable
func (registry *Registry) Pingable() bool {
	// Currently *.gcr.io/v2 can't be ping if users have each projects auth
	return !strings.HasSuffix(registry.URL, "gcr.io")
}

// Ping tries to contact a registry URL to make sure it is up and accessible.
func (registry *Registry) Ping(ctx context.Context) error {
	url := registry.url("/v2/")
	registry.Logf("registry.ping url=%s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := registry.Client.Do(req.WithContext(ctx))
	if resp != nil {
		defer resp.Body.Close()
	}
	return err
}
