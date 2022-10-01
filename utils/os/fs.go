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

package os

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/utils/os/fs"
)

func IsFileExist(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil || os.IsExist(err)
}

func IsCommandExist(cmdName string) bool {
	_, err := exec.LookPath(cmdName)
	return err == nil
}

func CountDirFiles(dirName string) int {
	var count int
	err := filepath.Walk(dirName, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		count++
		return nil
	})
	if err != nil {
		logrus.Warnf("failed to count dir files: %v", err)
		return 0
	}
	return count
}

func RecursionCopy(src, dst string) error {
	f := fs.NewFilesystem()
	if IsDir(src) {
		return f.CopyDir(src, dst)
	}
	err := f.MkdirAll(filepath.Dir(dst))
	if err != nil {
		return fmt.Errorf("failed to mkdir for recursion copy, err: %v", err)
	}

	_, err = f.CopyFile(src, dst)
	return err
}

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

type FilterOptions struct {
	All, OnlyDir, OnlyFile, WithFullPath bool
}

// GetDirNameListInDir :Get all Dir Name or file name List In Dir
func GetDirNameListInDir(dir string, opts FilterOptions) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var dirs []string

	if opts.All {
		for _, file := range files {
			if opts.WithFullPath {
				dirs = append(dirs, filepath.Join(dir, file.Name()))
			} else {
				dirs = append(dirs, file.Name())
			}
		}
		return dirs, nil
	}

	if opts.OnlyDir {
		for _, file := range files {
			if !file.IsDir() {
				continue
			}
			if opts.WithFullPath {
				dirs = append(dirs, filepath.Join(dir, file.Name()))
			} else {
				dirs = append(dirs, file.Name())
			}
		}
		return dirs, nil
	}

	if opts.OnlyFile {
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			if opts.WithFullPath {
				dirs = append(dirs, filepath.Join(dir, file.Name()))
			} else {
				dirs = append(dirs, file.Name())
			}
		}
		return dirs, nil
	}

	return dirs, nil
}

func IsAbs(hostPath string) bool {
	return path.IsAbs(hostPath) || filepath.IsAbs(hostPath)
}
