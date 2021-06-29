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

type PostCheckerService struct {
}

func (d *PostCheckerService) Run() error {
	checkerList, err := d.init()
	if err != nil {
		logger.Error(err)
		return err
	}
	/*	cluster, err := apply.GetCurrentCluster()
		if err != nil {
			return err
		}*/
	for _, checker := range checkerList {
		err = checker.Check()
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *PostCheckerService) init() ([]checker.Checker, error) {
	var checkerList []checker.Checker
	checkerList = append(checkerList, &checker.NodeChecker{}, &checker.PodChecker{}, &checker.SvcChecker{})
	return checkerList, nil
}

func NewPostCheckerService() CheckerService {
	return &PostCheckerService{}
}
