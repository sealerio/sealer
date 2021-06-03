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
	"fmt"
	"io/ioutil"
	"net/http"
)

type httpStatusError struct {
	Response *http.Response
	Body     []byte // Copied from `Response.Body` to avoid problems with unclosed bodies later. Nobody calls `err.Response.Body.Close()`, ever.
}

func (err *httpStatusError) Error() string {
	return fmt.Sprintf("http: non-successful response (status=%v body=%q)", err.Response.StatusCode, err.Body)
}

var _ error = &httpStatusError{}

// ErrorTransport defines the data structure for returning errors from the round tripper.
type ErrorTransport struct {
	Transport http.RoundTripper
}

// RoundTrip defines the round tripper for the error transport.
func (t *ErrorTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	resp, err := t.Transport.RoundTrip(request)
	if err != nil {
		return resp, err
	}

	if resp.StatusCode >= 500 || resp.StatusCode == http.StatusUnauthorized {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("http: failed to read response body (status=%v, err=%q)", resp.StatusCode, err)
		}

		return nil, &httpStatusError{
			Response: resp,
			Body:     body,
		}
	}

	return resp, err
}
