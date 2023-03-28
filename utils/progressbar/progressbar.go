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

package progressbar

import (
	"fmt"

	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
)

type EasyProgressUtil struct {
	progressbar.ProgressBar
}

var (
	width                  = 50
	optionEnableColorCodes = progressbar.OptionEnableColorCodes(true)
	optionSetWidth         = progressbar.OptionSetWidth(width)
	optionSetTheme         = progressbar.OptionSetTheme(progressbar.Theme{
		Saucer:        "=",
		SaucerHead:    ">",
		SaucerPadding: " ",
		BarStart:      "[",
		BarEnd:        "]",
	})
)

func NewEasyProgressUtil(total int, describe string) *EasyProgressUtil {
	return &EasyProgressUtil{
		*progressbar.NewOptions(total,
			optionEnableColorCodes,
			optionSetWidth,
			optionSetTheme,
			progressbar.OptionSetDescription(describe),
			// after finish, print a new line
			progressbar.OptionOnCompletion(func() {
				fmt.Println()
			}),
		),
	}
}

// increment add 1 to progress bar
func (epu *EasyProgressUtil) Increment() {
	if err := epu.Add(1); err != nil {
		logrus.Errorf("failed to increment progress bar, err: %s", err)
	}
}

// fail print error message
func (epu *EasyProgressUtil) Fail(err error) {
	if err != nil {
		epu.Describe(err.Error())
	}
}

// setTotal set total num of progress bar
func (epu *EasyProgressUtil) SetTotal(num int) {
	if num > epu.GetMax() {
		epu.ChangeMax(num)
	}
}
