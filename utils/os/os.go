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

// Interface collects system level operations
type Interface interface {
	MkdirAll(path string) error
	MkTmpdir() (string, error)
	CopyFile(src, dst string) (int64, error)
	CopyDir(srcPath, dstPath string) error
	RemoveAll(path ...string) error
	IsFileExist(fileName string) bool
	GetFilesSize(paths []string) (int64, error)
}

type FileReader interface {
	ReadLines() ([]string, error)
	ReadAll() ([]byte, error)
}

type FileWriter interface {
	WriteFile(content []byte) error
}
