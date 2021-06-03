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
	"net/http"
)

// CustomTransport defines the data structure for custom http.Request options.
type CustomTransport struct {
	Transport http.RoundTripper
	Headers   map[string]string
}

// RoundTrip defines the round tripper for the error transport.
func (t *CustomTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if len(t.Headers) != 0 {
		for header, value := range t.Headers {
			request.Header.Add(header, value)
		}
	}

	resp, err := t.Transport.RoundTrip(request)

	return resp, err
}
