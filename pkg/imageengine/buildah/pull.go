// Copyright © 2022 Alibaba Group Holding Ltd.
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

package buildah

import (
	"fmt"

	"os"

	"github.com/sealerio/sealer/pkg/define/options"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/pkg/auth"
	"github.com/pkg/errors"
)

func (engine *Engine) Pull(opts *options.PullOptions) error {
	if len(opts.Image) == 0 {
		return errors.Errorf("an image name must be specified")
	}

	if err := auth.CheckAuthFile(opts.Authfile); err != nil {
		return err
	}

	if err := engine.migratePullOptionsFlags2Command(opts); err != nil {
		return err
	}

	systemContext, err := parse.SystemContextFromOptions(engine.Command)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}
	systemContext.AuthFilePath = opts.Authfile

	store := engine.ImageStore()

	policy, ok := define.PolicyMap[opts.PullPolicy]
	if !ok {
		return fmt.Errorf("unsupported pull policy %q", opts.PullPolicy)
	}
	options := buildah.PullOptions{
		Store:         store,
		SystemContext: systemContext,
		// consider export this option later
		AllTags:      false,
		ReportWriter: os.Stderr,
		MaxRetries:   maxPullPushRetries,
		RetryDelay:   pullPushRetryDelay,
		PullPolicy:   policy,
	}

	if opts.Quiet {
		options.ReportWriter = nil // Turns off logging output
	}

	id, err := buildah.Pull(getContext(), opts.Image, options)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", id)
	return nil
}

func (engine *Engine) migratePullOptionsFlags2Command(opts *options.PullOptions) error {
	var (
		flags = engine.Command.Flags()
		err   error
	)

	if len(opts.Platform) > 0 {
		flags.StringSlice("platform", []string{}, "")
		// set pull platform, check "parse.SystemContextFromOptions(engine.Command)"
		err = flags.Set("platform", opts.Platform)
		if err != nil {
			return err
		}
	}
	return nil
}
