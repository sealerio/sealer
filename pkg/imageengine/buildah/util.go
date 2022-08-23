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

package buildah

import (
	"os"

	"path/filepath"

	"github.com/pkg/errors"
)

// DiscoverKubefile tries to find a Kubefile within the provided `path`.
func DiscoverKubefile(path string) (foundFile string, err error) {
	// Test for existence of the file
	target, err := os.Stat(path)
	if err != nil {
		return "", errors.Wrap(err, "discovering Kubefile")
	}

	switch mode := target.Mode(); {
	case mode.IsDir():
		// If the path is a real directory, we assume a Kubefile within it
		kubefile := filepath.Join(path, "Kubefile")

		// Test for existence of the Kubefile file
		file, err := os.Stat(kubefile)
		if err != nil {
			return "", errors.Wrap(err, "cannot find Kubefile in context directory")
		}

		// The file exists, now verify the correct mode
		if mode := file.Mode(); mode.IsRegular() {
			foundFile = kubefile
		} else {
			return "", errors.Errorf("assumed Kubefile %q is not a file", kubefile)
		}

	case mode.IsRegular():
		// If the context dir is a file, we assume this as Kubefile
		foundFile = path
	}

	return foundFile, nil
}
