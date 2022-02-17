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

package build

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/alibaba/sealer/build/buildkit/buildstorage"
	"github.com/alibaba/sealer/build/cloud"
	"github.com/alibaba/sealer/build/lite"
	"github.com/alibaba/sealer/build/local"
	"github.com/alibaba/sealer/common"
)

var ProviderMap = map[string]string{
	common.LocalBuild:     common.BAREMETAL,
	common.AliCloudBuild:  common.AliCloud,
	common.ContainerBuild: common.CONTAINER,
}

func NewLocalBuilder(config *Config) (Interface, error) {
	return &local.Builder{
		BuildType: config.BuildType,
		NoCache:   config.NoCache,
		NoBase:    config.NoBase,
		BuildArgs: config.BuildArgs,
	}, nil
}

func NewCloudBuilder(config *Config) (Interface, error) {
	provider := common.AliCloud
	if config.BuildType != "" {
		provider = ProviderMap[config.BuildType]
	}

	return &cloud.Builder{
		BuildType:          config.BuildType,
		NoCache:            config.NoCache,
		NoBase:             config.NoBase,
		BuildArgs:          config.BuildArgs,
		Provider:           provider,
		TmpClusterFilePath: common.TmpClusterfile,
	}, nil
}

func NewLiteBuilder(config *Config) (Interface, error) {
	storage, err := parseOutputStorage(config.Output)
	if err != nil {
		return nil, err
	}
	return &lite.Builder{
		BuildType:     config.BuildType,
		NoCache:       config.NoCache,
		NoBase:        config.NoBase,
		BuildArgs:     config.BuildArgs,
		StorageDriver: storage,
	}, nil
}

func parseOutputStorage(s string) (buildstorage.StorageDriver, error) {
	sd := buildstorage.StorageDriver{
		DriverType: buildstorage.FileSystemFactory,
		Parameters: map[string]string{},
	}
	if s == "" {
		return sd, nil
	}

	csvReader := csv.NewReader(strings.NewReader(s))
	fields, err := csvReader.Read()
	if err != nil {
		return sd, err
	}
	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			return sd, fmt.Errorf("invalid value %s", field)
		}
		key := strings.ToLower(parts[0])
		value := parts[1]
		switch key {
		case "type":
			sd.DriverType = value
		default:
			sd.Parameters[key] = value
		}
	}

	// filesystem type do not need dest parameter.
	if sd.DriverType == buildstorage.FileSystemFactory {
		return sd, nil
	}

	if _, ok := sd.Parameters["dest"]; !ok {
		return sd, fmt.Errorf("--output requires dest=<dest>")
	}

	return sd, nil
}
