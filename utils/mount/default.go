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

package mount

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/alibaba/sealer/logger"

	"github.com/alibaba/sealer/utils"
)

type Default struct {
}

// Unmount target
func (d *Default) Unmount(target string) error {
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("remote target failed: %s", err)
	}
	return nil
}

// copy all layers to target merged dir
func (d *Default) Mount(target string, upperDir string, layers ...string) error {
	//if target is empty,return err
	if target == "" {
		return fmt.Errorf("target is empty")
	}

	utils.Reverse(layers)

	for _, layer := range layers {
		srcInfo, err := os.Stat(layer)
		if err != nil {
			return fmt.Errorf("get srcInfo err: %s", err)
		}
		if srcInfo.IsDir() {
			err := copyDir(layer, target)
			if err != nil {
				return fmt.Errorf("copyDir [%s] to [%s] failed: %s", layer, target, err)
			}
		} else {
			IsExist, err := PathExists(target)
			if err != nil {
				return err
			}
			if !IsExist {
				err = os.Mkdir(target, 0666)
				if err != nil {
					return fmt.Errorf("mkdir [%s] error %v", target, err)
				}
			}
			_file := filepath.Base(layer)
			dst := path.Join(target, _file)
			err = copyFile(layer, dst)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func copyDir(srcPath string, dstPath string) error {
	IsExist, err := PathExists(dstPath)
	if err != nil {
		return err
	}
	if !IsExist {
		err = os.Mkdir(dstPath, 0666)
		if err != nil {
			return fmt.Errorf("mkdir [%s] error %v", dstPath, err)
		}
	}

	srcFiles, err := ioutil.ReadDir(srcPath)
	if err != nil {
		return err
	}
	for _, file := range srcFiles {
		src := path.Join(srcPath, file.Name())
		dst := path.Join(dstPath, file.Name())
		if file.IsDir() {
			err = copyDir(src, dst)
			if err != nil {
				return err
			}
		} else {
			err = copyFile(src, dst)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	// open srcfile
	srcFile, err := os.Open(filepath.Clean(src))
	if err != nil {
		return fmt.Errorf("open file [%s] failed: %s", src, err)
	}
	defer func() {
		if err := srcFile.Close(); err != nil {
			logger.Fatal("failed to close file")
		}
	}()
	// create dstfile
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create file err: %s", err)
	}
	defer func() {
		if err := dstFile.Close(); err != nil {
			logger.Fatal("failed to close file")
		}
	}()

	// copy  file
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("copy file err: %s", err)
	}
	return nil
}

//notExist false ,Exist true
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("os.Stat(%s) err: %s", path, err)
}
