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
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/ssh"
)

type PreApplyCheckerService struct {
	currentCluster, desiredCluster *v1.Cluster
}

func (d *PreApplyCheckerService) Run() error {
	checkerList, err := d.init()
	if err != nil {
		logger.Error(err)
		return err
	}
	for _, Checker := range checkerList {
		err = Checker.Check()
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *PreApplyCheckerService) init() ([]checker.Checker, error) {
	var checkerList []checker.Checker
	SSH := ssh.NewSSHByCluster(d.desiredCluster)
	hostCheckIPList := append(d.desiredCluster.Spec.Masters.IPList, d.desiredCluster.Spec.Nodes.IPList...)
	checkerList = append(checkerList, checker.NewApplyChecker(d.currentCluster, d.desiredCluster), checker.NewHostChecker(SSH, hostCheckIPList))
	return checkerList, nil
}

func NewPreApplyCheckerService(currentCluster, desiredCluster *v1.Cluster) CheckerService {
	return &PreApplyCheckerService{currentCluster, desiredCluster}
}
