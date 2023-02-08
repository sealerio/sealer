// Copyright © 2022 Alibaba Group Holding Ltd.
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

package parser

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func getFileFromURL(src, rename, mountPoint string) (filePath string, err error) {
	url, err := url.Parse(src)
	if err != nil {
		return "", err
	}
	response, err := http.Get(src) /* #nosec G107 */
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http reader")
		}
	}(response.Body)
	// Figure out what to name the new content.
	name := rename
	if name == "" {
		name = path.Base(url.Path)
	}
	target := filepath.Clean(filepath.Join(mountPoint, name))
	f, err := os.Create(target)
	if err != nil {
		return "", errors.Wrapf(err, "error creating file to target %s for %s", target, src)
	}
	defer func() {
		_ = f.Close()
	}()
	_, err = io.Copy(f, response.Body)
	if err != nil {
		return "", errors.Wrapf(err, "error writing %q to temporary file %q", src, f.Name())
	}
	return target, nil
}
