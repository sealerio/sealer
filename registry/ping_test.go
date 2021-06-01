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
	"testing"
)

func TestPingable(t *testing.T) {
	testcases := map[string]struct {
		registry Registry
		expect   bool
	}{
		"Docker": {
			registry: Registry{URL: "https://registry-1.docker.io"},
			expect:   true,
		},
		"GCR_global": {
			registry: Registry{URL: "https://gcr.io"},
			expect:   false,
		},
		"GCR_asia": {
			registry: Registry{URL: "https://asia.gcr.io"},
			expect:   false,
		},
	}
	for label, testcase := range testcases {
		actual := testcase.registry.Pingable()
		if testcase.expect != actual {
			t.Fatalf("%s: expected (%v), got (%v)", label, testcase.expect, actual)
		}
	}
}
