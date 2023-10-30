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

package preflight

import (
	"bytes"
	"fmt"

	"github.com/sealerio/sealer/utils/strings"
	"github.com/sirupsen/logrus"
)

// OptionFunc configures a runner list
type OptionFunc func(*RunOptions)

type RunOptions struct {
	// SkipList: if checker name in this list, will skip to check. default is lowercase.
	SkipList []string

	// IgnoreList: if checker name in this list, will ignore its error after run check. default is lowercase.
	IgnoreList []string
}

func WithSkips(skips []string) OptionFunc {
	return func(o *RunOptions) {
		o.SkipList = skips
	}
}

func WithIgnores(ignores []string) OptionFunc {
	return func(o *RunOptions) {
		o.IgnoreList = ignores
	}
}

type Runner struct {
	Checkers []Checker
}

// Execute checker validate and dispatch to different results
func (r *Runner) Execute(optFunc ...OptionFunc) error {
	var options RunOptions
	for _, opt := range optFunc {
		opt(&options)
	}

	var errsBuffer bytes.Buffer
	for _, checker := range r.Checkers {
		// run the validation
		name := checker.Name()

		// skip specified checker
		if len(options.SkipList) > 0 && strings.IsInSlice(name, options.SkipList) {
			continue
		}

		warnings, errs := checker.Check()

		// ignore check error and append it to warnings
		if strings.IsInSlice(name, options.IgnoreList) {
			warnings = append(warnings, errs...)
			errs = []error{}
		}

		for _, w := range warnings {
			logrus.Warnf(fmt.Sprintf("\t[WARNING %s]: %v\n", name, w))
		}

		for _, i := range errs {
			errsBuffer.WriteString(fmt.Sprintf("\t[ERROR %s]: %v\n", name, i.Error()))
		}
	}

	if errsBuffer.Len() > 0 {
		return fmt.Errorf("prechecked some error:%s", errsBuffer.String())
	}

	return nil
}
