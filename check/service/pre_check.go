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

package service

import (
	"github.com/alibaba/sealer/check/checker"
	"github.com/alibaba/sealer/logger"
)

type PreCheckerService struct {
}

func (p *PreCheckerService) Run() error {
	checkerList, err := p.init()
	if err != nil {
		logger.Error(err)
		return err
	}

	for _, c := range checkerList {
		err = c.Check()
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PreCheckerService) init() ([]checker.PreChecker, error) {
	var checkerList []checker.PreChecker
	checkerList = append(checkerList, &checker.RegistryChecker{})
	return checkerList, nil
}
