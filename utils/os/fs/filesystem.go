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

package fs

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

var FS = NewFilesystem()

type Interface interface {
	Stat(name string) (os.FileInfo, error)
	Rename(oldPath, newPath string) error
	MkdirAll(path string) error
	MkTmpdir(path string) (string, error)
	CopyFile(src, dst string) (int64, error)
	CopyDir(srcPath, dstPath string) error
	RemoveAll(path ...string) error
	GetFilesSize(paths []string) (int64, error)
}

type filesystem struct{}

func (f filesystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (f filesystem) Rename(oldPath, newPath string) error {
	// remove newPath before mv files to target
	_, err := f.Stat(newPath)
	if err == nil {
		err = f.RemoveAll(newPath)
		if err != nil {
			return err
		}
	}

	// create dir if filepath.Dir(newPath) not exist
	_, err = f.Stat(filepath.Dir(newPath))
	if err != nil {
		err = f.MkdirAll(filepath.Dir(newPath))
		if err != nil {
			return err
		}
	}

	return os.Rename(oldPath, newPath)
}

func (f filesystem) RemoveAll(path ...string) error {
	for _, fi := range path {
		err := os.RemoveAll(fi)
		if err != nil {
			return fmt.Errorf("failed to clean file %s: %v", fi, err)
		}
	}
	return nil
}

func (f filesystem) MkdirAll(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

func (f filesystem) MkTmpdir(path string) (string, error) {
	tempDir, err := os.MkdirTemp(path, ".DTmp-")
	if err != nil {
		return "", err
	}
	return tempDir, os.MkdirAll(tempDir, os.ModePerm)
}

func (f filesystem) CopyDir(srcPath, dstPath string) error {
	err := f.MkdirAll(dstPath)
	if err != nil {
		return err
	}

	fis, err := os.ReadDir(srcPath)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		src := filepath.Join(srcPath, fi.Name())
		dst := filepath.Join(dstPath, fi.Name())
		if fi.IsDir() {
			err = f.CopyDir(src, dst)
			if err != nil {
				return err
			}
		} else {
			_, err = f.CopyFile(src, dst)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (f filesystem) CopyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	header, err := tar.FileInfoHeader(sourceFileStat, src)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info header for %s, err: %v", src, err)
	}

	if sourceFileStat.Mode()&os.ModeCharDevice != 0 && header.Devminor == 0 && header.Devmajor == 0 {
		err = unix.Mknod(dst, unix.S_IFCHR, 0)
		if err != nil {
			return 0, err
		}
		return 0, os.Chown(dst, header.Uid, header.Gid)
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(filepath.Clean(src))
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := source.Close(); err != nil {
			logrus.Errorf("failed to close file: %v", err)
		}
	}()
	//will overwrite dst when dst is existed
	destination, err := os.Create(filepath.Clean(dst))
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := destination.Close(); err != nil {
			logrus.Errorf("failed to close file: %v", err)
		}
	}()
	err = destination.Chmod(sourceFileStat.Mode())
	if err != nil {
		return 0, err
	}

	err = os.Chown(dst, header.Uid, header.Gid)
	if err != nil {
		return 0, err
	}
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func (f filesystem) GetFilesSize(paths []string) (int64, error) {
	var size int64
	for i := range paths {
		s, err := f.getFileSize(paths[i])
		if err != nil {
			return 0, err
		}
		size += s
	}
	return size, nil
}

func (f filesystem) getFileSize(path string) (size int64, err error) {
	_, err = os.Stat(path)
	if err != nil {
		return
	}
	err = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func NewFilesystem() Interface {
	return filesystem{}
}
