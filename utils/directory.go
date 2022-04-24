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

package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func Sub(dir1, dir2 string) ([]string, error) {
	var deleted []string
	isSame, err := IsSameDir(dir1, dir2)
	if err != nil {
		return nil, err
	}
	if isSame {
		deleted = append(deleted, dir1)
		return deleted, nil
	}

	contents, err := ioutil.ReadDir(dir1)
	if err != nil {
		return nil, err
	}

	for _, file := range contents {
		if !IsExist(filepath.Join(dir2, file.Name())) {
			continue
		}

		if file.IsDir() {
			data, err := Sub(filepath.Join(dir1, file.Name()), filepath.Join(dir2, file.Name()))
			if err != nil {
				return nil, err
			}
			deleted = append(deleted, data...)
		} else {
			deleted = append(deleted, filepath.Join(dir1, file.Name()))
		}
	}
	return deleted, nil
}

func IsSameDir(dir1, dir2 string) (bool, error) {
	list1, err := getFileList(dir1)
	if err != nil {
		return false, err
	}

	list2, err := getFileList(dir2)
	if err != nil {
		return false, err
	}
	add, sub := GetDiffHosts(list1, list2)
	if len(add) == 0 && len(sub) == 0 {
		return true, nil
	}

	return false, nil
}

func getFileList(path string) ([]string, error) {
	var directory []string
	var err error

	walkFn := func(currPath string, info os.FileInfo, err error) error {
		newContent := strings.TrimPrefix(currPath, path)
		if newContent != "" {
			directory = append(directory, newContent)
		}
		return nil
	}

	err = filepath.Walk(path, walkFn)

	return directory, err
}
