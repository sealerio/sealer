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

package weed

import (
	"errors"
	"io"
	"net/http"
	"os"
	"runtime"
)

var (
	weedURL = "https://github.com/seaweedfs/seaweedfs/releases/download/3.54/"
)

const (
	extractFolder = "/tmp"
)

func weedDownloadURL() (string, error) {
	if runtime.GOOS != "linux" {
		return "", errors.New("unsupported os")
	}
	switch arch := runtime.GOARCH; arch {
	case "amd64":
		weedURL += "linux_amd64.tar.gz"
	case "arm64":
		weedURL += "linux_arm.tar.gz"
	default:
		return "", errors.New("unsupported arch")
	}
	return weedURL, nil
}

func downloadFile(url string, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// check if the destination folder exists
	_, err = os.Stat(dest)
	if err == nil {
		_ = os.RemoveAll(dest)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
