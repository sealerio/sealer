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

package check

import (
	"github.com/alibaba/sealer/check/service"
)

const (
	PreCheck      = "PreCheck"
	PreApplyCheck = "PreApplyCheck"
	PostCheck     = "PostCheck"
)

type Args struct {
	PreApplyChecker *service.PreApplyCheckerService
}

func NewChecker(args Args, checkerArgs CheckerArgs) (CheckerService, error) {
	switch checkerArgs {
	case PreCheck:
		return &service.PreCheckerService{}, nil
	case PostCheck:
		return &service.PostCheckerService{}, nil
	case PreApplyCheck:
		return &service.PreApplyCheckerService{Desired: args.PreApplyChecker.Desired, Current: args.PreApplyChecker.Current}, nil
	default:
		return &service.DefaultCheckerService{}, nil
	}
}
