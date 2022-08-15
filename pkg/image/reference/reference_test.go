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

package reference

import (
	"fmt"
	"testing"
)

func TestParseToNamed(t *testing.T) {
	type namedTest struct {
		name    string
		desired Named
	}

	ts := []namedTest{
		{
			name: "xxx.com/abc/tag:v1",
			desired: Named{
				raw:     "xxx.com/abc/tag:v1",
				domain:  "xxx.com",
				repo:    "abc/tag",
				tag:     "v1",
				repoTag: "abc/tag:v1",
			},
		},
		{
			name: "abc/tag:v1",
			desired: Named{
				raw:     "abc/tag:v1",
				domain:  defaultDomain,
				repo:    "abc/tag",
				tag:     "v1",
				repoTag: "abc/tag:v1",
			},
		},
		{
			name: "tag:v1",
			desired: Named{
				raw:     "tag:v1",
				domain:  defaultDomain,
				repo:    defaultRepo + "/tag",
				tag:     "v1",
				repoTag: defaultRepo + "/tag:v1",
			},
		},
		{
			name: "tag",
			desired: Named{
				raw:     "tag:" + defaultTag,
				domain:  defaultDomain,
				repo:    defaultRepo + "/tag",
				tag:     defaultTag,
				repoTag: defaultRepo + "/tag:" + defaultTag,
			},
		},
		{
			name: "xxx.com:5000/abc/tag",
			desired: Named{
				raw:     "xxx.com:5000/abc/tag:" + defaultTag,
				domain:  "xxx.com:5000",
				repo:    "abc/tag",
				tag:     defaultTag,
				repoTag: "abc/tag:" + defaultTag,
			},
		},
	}

	for _, tt := range ts {
		named, err := ParseToNamed(tt.name)
		if err != nil {
			t.Fatalf(err.Error())
		}
		err = compareNamed(named, tt.desired)
		if err != nil {
			t.Fatalf(err.Error())
		}
	}
}

func compareNamed(a, b Named) error {
	type compare struct {
		c, d string
	}
	cs := []compare{{
		c: a.raw,
		d: b.raw,
	}, {
		c: a.tag,
		d: b.tag,
	}, {
		c: a.repoTag,
		d: b.repoTag,
	}, {
		c: a.repo,
		d: b.repo,
	}, {
		c: a.domain,
		d: b.domain,
	}}
	for _, c := range cs {
		if c.d != c.c {
			return fmt.Errorf("%s does not equal to %s", c.c, c.d)
		}
	}
	return nil
}
