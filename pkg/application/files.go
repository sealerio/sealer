// Copyright © 2023 Alibaba Group Holding Ltd.
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

package application

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/imdario/mergo"
	"github.com/sealerio/sealer/pkg/env"
	v2 "github.com/sealerio/sealer/types/api/v2"
	osUtils "github.com/sealerio/sealer/utils/os"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func newFileProcessor(appFile v2.AppFile) (FileProcessor, error) {
	switch appFile.Strategy {
	case v2.OverWriteStrategy:
		return overWriteProcessor{appFile}, nil
	case v2.MergeStrategy:
		return mergeProcessor{appFile}, nil
	}

	return nil, fmt.Errorf("failed to init fileProcessor,%s is not register", appFile.Strategy)
}

// overWriteProcessor :this will overwrite the FilePath with the Values.
type overWriteProcessor struct {
	v2.AppFile
}

func (r overWriteProcessor) Process(appRoot string) error {
	target := filepath.Join(appRoot, r.Path)

	logrus.Debugf("will do overwrite processor on the file : %s", target)
	err := osUtils.NewCommonWriter(target).WriteFile([]byte(r.Data))
	if err != nil {
		return fmt.Errorf("failed to write to file %s with raw mode: %v", target, err)
	}
	return nil
}

// mergeProcessor :this will merge the FilePath with the Values.
// Only files in yaml format are supported.
// if Strategy is "merge" will deeply merge each yaml file section.
type mergeProcessor struct {
	v2.AppFile
}

func (m mergeProcessor) Process(appRoot string) error {
	var (
		result     [][]byte
		srcDataMap = make(map[string]interface{})
	)

	err := yaml.Unmarshal([]byte(m.Data), &srcDataMap)
	if err != nil {
		return fmt.Errorf("failed to load config data: %v", err)
	}

	target := filepath.Join(appRoot, m.Path)

	logrus.Debugf("will do merge processor on the file : %s", target)

	contents, err := os.ReadFile(filepath.Clean(target))
	if err != nil {
		return err
	}

	for _, section := range bytes.Split(contents, []byte("---\n")) {
		destDataMap := make(map[string]interface{})

		err = yaml.Unmarshal(section, &destDataMap)
		if err != nil {
			return fmt.Errorf("failed to unmarshal config data: %v", err)
		}

		err = mergo.Merge(&destDataMap, &srcDataMap, mergo.WithOverride)
		if err != nil {
			return fmt.Errorf("failed to merge config: %v", err)
		}

		out, err := yaml.Marshal(destDataMap)
		if err != nil {
			return err
		}

		result = append(result, out)
	}

	err = osUtils.NewCommonWriter(target).WriteFile(bytes.Join(result, []byte("---\n")))
	if err != nil {
		return fmt.Errorf("failed to write to file %s with raw mode: %v", target, err)
	}
	return nil
}

// envRender :this will render the FilePath with the Values.
type envRender struct {
	envData map[string]string
}

func (e envRender) Process(appRoot string) error {
	if len(e.envData) == 0 {
		return nil
	}

	logrus.Debugf("will render the dir : %s with the values: %+v\n", appRoot, e.envData)

	return env.RenderTemplate(appRoot, e.envData)
}
