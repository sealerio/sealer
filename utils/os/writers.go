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
	"os"
	"path/filepath"

	"github.com/sealerio/sealer/common"

	"github.com/sirupsen/logrus"
)

type FileWriter interface {
	WriteFile(content []byte) error
}

type atomicFileWriter struct {
	f    *os.File
	path string
	perm os.FileMode
}

func (a *atomicFileWriter) close() (err error) {
	if err = a.f.Sync(); err != nil {
		err := a.f.Close()
		if err != nil {
			return err
		}
		return err
	}
	if err := a.f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(a.f.Name(), a.perm); err != nil {
		return err
	}
	return os.Rename(a.f.Name(), a.path)
}

func newAtomicFileWriter(path string, perm os.FileMode) (*atomicFileWriter, error) {
	tmpFile, err := os.CreateTemp(filepath.Dir(path), ".FTmp-")
	if err != nil {
		return nil, err
	}
	return &atomicFileWriter{f: tmpFile, path: path, perm: perm}, nil
}

type atomicWriter struct{ fileName string }

func (a atomicWriter) Clean(file *os.File) {
	if file == nil {
		return
	}
	// the following operation won't fail regularly, if failed, log it
	err := file.Close()
	if err != nil && err != os.ErrClosed {
		logrus.Warn(err)
	}
	err = os.Remove(file.Name())
	if err != nil {
		logrus.Warn(err)
	}
}

func (a atomicWriter) WriteFile(content []byte) error {
	dir := filepath.Dir(a.fileName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, common.FileMode0755); err != nil {
			return err
		}
	}

	afw, err := newAtomicFileWriter(a.fileName, common.FileMode0644)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			a.Clean(afw.f)
		}
	}()
	if _, err = afw.f.Write(content); err != nil {
		return err
	}
	return afw.close()
}

func NewAtomicWriter(fileName string) FileWriter {
	return atomicWriter{
		fileName: fileName,
	}
}

type commonWriter struct {
	fileName string
}

func (c commonWriter) WriteFile(content []byte) error {
	dir := filepath.Dir(c.fileName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, common.FileMode0755); err != nil {
			return err
		}
	}
	return os.WriteFile(c.fileName, content, common.FileMode0644)
}

func NewCommonWriter(fileName string) FileWriter {
	return commonWriter{
		fileName: fileName,
	}
}
