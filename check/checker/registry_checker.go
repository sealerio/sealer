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
	"fmt"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/utils"
)

type RegistryChecker struct {
}

func (r *RegistryChecker) Check() error {
	// check registry info;
	authFile := common.DefaultRegistryAuthConfigDir()
	if !utils.IsFileExist(authFile) {
		return fmt.Errorf("registry auth info not found,please run 'sealer login' first")
	}
	return nil
}
