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

package checker

import (
	"context"
	"fmt"
	"github.com/alibaba/sealer/pkg/image/distributionutil"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/utils"
)

type RegistryChecker struct {
	RegistryDomain string
}

func (r *RegistryChecker) Check() error {
	// check the existence of the docker.json ;
	authFile := common.DefaultRegistryAuthConfigDir()
	if !utils.IsFileExist(authFile) {
		return fmt.Errorf("registry auth info not found,please run 'sealer login' first")
	}
	// try to login with auth info
	authConfig, err := utils.GetDockerAuthInfoFromDocker(r.RegistryDomain)
	if err != nil {
		return fmt.Errorf("failed to get auth info, err: %s", err)
	}

	err = distributionutil.Login(context.Background(), &authConfig)
	if err != nil {
		return fmt.Errorf("%v authentication failed", r.RegistryDomain)
	}
	return nil
}

func NewRegistryChecker(registryDomain string) Checker {
	return &RegistryChecker{RegistryDomain: registryDomain}
}
