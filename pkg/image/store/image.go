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
	"path/filepath"

	"github.com/alibaba/sealer/common"
	pkgutils "github.com/alibaba/sealer/utils"
)

func DeleteImageLocal(imageID string) (err error) {
	return deleteImage(imageID)
}

func deleteImage(imageID string) error {
	file := filepath.Join(common.DefaultImageDBRootDir, imageID+common.YamlSuffix)
	if pkgutils.IsFileExist(file) {
		err := pkgutils.CleanFiles(file)
		if err != nil {
			return err
		}
	}
	return nil
}
