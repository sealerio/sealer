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

package boot

import (
	"fmt"
	"os"

	"github.com/sealerio/sealer/logger"

	"github.com/sealerio/sealer/common"
)

var rootDirs = []string{
	logger.DefaultLogDir,
	common.DefaultTmpDir,
	common.DefaultImageRootDir,
	common.DefaultImageMetaRootDir,
	common.DefaultImageDBRootDir,
	common.DefaultLayerDir,
	common.DefaultLayerDBRoot}

func initRootDirectory() error {
	for _, dir := range rootDirs {
		err := os.MkdirAll(dir, common.FileMode0755)
		if err != nil {
			return fmt.Errorf("failed to mkdir %s, err: %s", dir, err)
		}
	}
	return nil
}

func OnBoot() error {
	return initRootDirectory()
}
