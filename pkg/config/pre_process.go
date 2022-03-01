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

package config

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/wonderivan/logger"
	yaml2 "gopkg.in/yaml.v2"
	"sigs.k8s.io/yaml"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

type PreProcessor interface {
	Process(config *v1.Config) error
	Name() string
}

func NewProcessorsAndRun(config *v1.Config) error {
	pmap := make(map[string]PreProcessor)
	valueProcessor := &valueProcessor{}
	toJsonProcessor := &toJsonProcessor{}
	toBase64Processor := &toBase64Processor{}

	pmap[valueProcessor.Name()] = valueProcessor
	pmap[toJsonProcessor.Name()] = toJsonProcessor
	pmap[toBase64Processor.Name()] = toBase64Processor

	processors := strings.Split(config.Spec.Process, "|")
	for _, pname := range processors {
		prossor, ok := pmap[pname]
		if !ok {
			logger.Warn("not found config processor: %s", pname)
		}
		if err := prossor.Process(config); err != nil {
			return err
		}
	}

	return nil
}

type valueProcessor struct{}

func (v valueProcessor) Process(config *v1.Config) error {
	config.Labels = make(map[string]string)
	config.Labels["preprocess.value"] = "true"
	return nil
}

func (v valueProcessor) Name() string {
	return "value"
}

type toJsonProcessor struct{}

func (t toJsonProcessor) Process(config *v1.Config) error {
	if v, ok := config.Labels["preprocess.value"]; !ok || v != "true" {
		json, err := yaml.YAMLToJSON([]byte(config.Spec.Data))
		if err != nil {
			return fmt.Errorf("failed to resolve config data to json, %v", err)
		}
		config.Spec.Data = string(json)
		return nil
	}

	dataMap := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(config.Spec.Data), &dataMap)
	if err != nil {
		return fmt.Errorf("convert yaml data to map failed, %v", err)
	}

	for k, v := range dataMap {
		data, err := yaml.Marshal(v)
		if err != nil {
			return fmt.Errorf("encode yaml failed,%v", err)
		}

		bytes, err := yaml.YAMLToJSON(data)
		if err != nil {
			return fmt.Errorf("toJson: failed to convert yaml to json, key is %s, %v", k, err)
		}
		dataMap[k] = string(bytes)
	}

	data, err := yaml2.Marshal(dataMap)
	if err != nil {
		return fmt.Errorf("failed to convert data map, %v,%v", dataMap, err)
	}
	config.Spec.Data = string(data)

	return nil
}

func (t toJsonProcessor) Name() string {
	return "toJson"
}

type toBase64Processor struct{}

func (t toBase64Processor) Process(config *v1.Config) error {
	if v, ok := config.Labels["preprocess.value"]; !ok || v != "true" {
		config.Spec.Data = base64.StdEncoding.EncodeToString([]byte(config.Spec.Data))
		return nil
	}

	dataMap := make(map[string]string)
	err := yaml.Unmarshal([]byte(config.Spec.Data), &dataMap)
	if err != nil {
		return fmt.Errorf("tobase64: convert yaml data to map failed, %v", err)
	}

	for k, v := range dataMap {
		dataMap[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}
	bs, err := yaml.Marshal(dataMap)
	if err != nil {
		fmt.Errorf("failed to convert base64 to yaml, %v", err)
	}

	config.Spec.Data = string(bs)

	return nil
}

func (t toBase64Processor) Name() string {
	return "toBase64"
}
