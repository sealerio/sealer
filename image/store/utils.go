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

package store

import (
	"io/ioutil"
	"path/filepath"
)

//var supportedDigestAlgo = map[string]bool{
//	digest.SHA256.String(): true,
//	digest.SHA384.String(): true,
//	digest.SHA512.String(): true,
//}

func getDirListInDir(dir string) ([]string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, file := range files {
		// avoid adding some other dirs created by users
		if file.IsDir() {
			dirs = append(dirs, filepath.Join(dir, file.Name()))
		}
	}
	return dirs, nil
}

func (ls LayerStorage) traverseLayerDB() ([]string, error) {
	// TODO maybe there no need to traverse layerdb, just clarify how many sha supported in a list
	shaDirs, err := getDirListInDir(ls.LayerDBRoot)
	if err != nil {
		return nil, err
	}

	var layerDirs []string
	for _, shaDir := range shaDirs {
		layerDirList, err := getDirListInDir(shaDir)
		if err != nil {
			return nil, err
		}
		layerDirs = append(layerDirs, layerDirList...)
	}
	return layerDirs, nil
}
