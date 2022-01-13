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

package cloud

import (
	"fmt"
	"io"
	"os"

	"github.com/alibaba/sealer/logger"

	"github.com/alibaba/sealer/utils/archive"
)

func tarBuildContext(kubeFilePath string, context string, tarFileName string) error {
	file, err := os.Create(tarFileName)
	if err != nil {
		return fmt.Errorf("failed to create %s, err: %v", tarFileName, err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			logger.Fatal("failed to close file")
		}
	}()

	var pathsToCompress []string
	pathsToCompress = append(pathsToCompress, kubeFilePath, context)
	tarReader, err := archive.TarWithoutRootDir(pathsToCompress...)
	if err != nil {
		return fmt.Errorf("failed to new tar reader when send build context, err: %v", err)
	}
	defer tarReader.Close()

	_, err = io.Copy(file, tarReader)
	if err != nil {
		return fmt.Errorf("failed to tar build context, err: %v", err)
	}
	return nil
}
