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
	"strings"
	"testing"
)

type authServiceMock struct {
	service string
	realm   string
	scope   []string
}

type challengeTestCase struct {
	header      string
	errorString string
	value       authServiceMock
}

func (asm authServiceMock) equalTo(v *authService) bool {
	if asm.service != v.Service {
		return false
	}
	for i, v := range v.Scope {
		if v != asm.scope[i] {
			return false
		}
	}

	return asm.realm == v.Realm.String()
}

func TestParseChallenge(t *testing.T) {
	challengeHeaderCases := []challengeTestCase{
		{
			header: `Bearer realm="https://foobar.com/api/v1/token",service=foobar.com,scope=""`,
			value: authServiceMock{
				service: "foobar.com",
				realm:   "https://foobar.com/api/v1/token",
			},
		},
		{
			header: `Bearer realm="https://r.j3ss.co/auth",service="Docker registry",scope="repository:chrome:pull"`,
			value: authServiceMock{
				service: "Docker registry",
				realm:   "https://r.j3ss.co/auth",
				scope:   []string{"repository:chrome:pull"},
			},
		},
		{
			header:      `Basic realm="https://r.j3ss.co/auth",service="Docker registry"`,
			errorString: "basic auth required",
		},
		{
			header:      `Basic realm="Registry Realm",service="Docker registry"`,
			errorString: "basic auth required",
		},
	}

	for _, tc := range challengeHeaderCases {
		val, err := parseChallenge(tc.header)
		if err != nil && !strings.Contains(err.Error(), tc.errorString) {
			t.Fatalf("expected error to contain %v,  got %s", tc.errorString, err)
		}
		if err == nil && !tc.value.equalTo(val) {
			t.Fatalf("got %v, expected %v", val, tc.value)
		}
	}
}

func TestParseChallengePush(t *testing.T) {
	challengeHeaderCases := []challengeTestCase{
		{
			header: `Bearer realm="https://foo.com/v2/token",service="foo.com",scope="repository:pdr/tls:pull,push"`,
			value: authServiceMock{
				realm:   "https://foo.com/v2/token",
				service: "foo.com",
				scope:   []string{"repository:pdr/tls:pull,push"},
			},
		},
	}
	for _, tc := range challengeHeaderCases {
		val, err := parseChallenge(tc.header)
		if err != nil && !strings.Contains(err.Error(), tc.errorString) {
			t.Fatalf("expected error to contain %v,  got %s", tc.errorString, err)
		}
		if !tc.value.equalTo(val) {
			t.Fatalf("got %v, expected %v", val, tc.value)
		}
	}
}
