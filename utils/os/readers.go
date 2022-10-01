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
	"bufio"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type FileReader interface {
	ReadLines() ([]string, error)
	ReadAll() ([]byte, error)
}

type fileReader struct {
	fileName string
}

func (r fileReader) ReadLines() ([]string, error) {
	var lines []string

	if _, err := os.Stat(r.fileName); err != nil || os.IsNotExist(err) {
		return nil, errors.New("no such file")
	}

	file, err := os.Open(filepath.Clean(r.fileName))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logrus.Fatal("failed to close file")
		}
	}()
	br := bufio.NewReader(file)
	for {
		line, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		lines = append(lines, string(line))
	}
	return lines, nil
}

func (r fileReader) ReadAll() ([]byte, error) {
	if _, err := os.Stat(r.fileName); err != nil || os.IsNotExist(err) {
		return nil, errors.New("no such file")
	}

	file, err := os.Open(filepath.Clean(r.fileName))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logrus.Errorf("failed to close file: %v", err)
		}
	}()

	content, err := os.ReadFile(filepath.Clean(r.fileName))
	if err != nil {
		return nil, err
	}

	return content, nil
}

func NewFileReader(fileName string) FileReader {
	return fileReader{
		fileName: fileName,
	}
}
