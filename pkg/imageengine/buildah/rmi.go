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
	"context"
	"fmt"

	"github.com/sealerio/sealer/pkg/define/options"

	"github.com/containers/common/libimage"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

func (engine *Engine) RemoveImage(opts *options.RemoveImageOptions) error {
	if len(opts.ImageNamesOrIDs) == 0 && !opts.All && !opts.Prune {
		return errors.Errorf("image name or ID must be specified")
	}
	if len(opts.ImageNamesOrIDs) > 0 && opts.All {
		return errors.Errorf("when using the --all switch, you may not pass any images names or IDs")
	}
	if opts.All && opts.Prune {
		return errors.Errorf("when using the --all switch, you may not use --prune switch")
	}
	if len(opts.ImageNamesOrIDs) > 0 && opts.Prune {
		return errors.Errorf("when using the --prune switch, you may not pass any images names or IDs")
	}

	options := &libimage.RemoveImagesOptions{
		Filters: []string{"readonly=false"},
	}

	if opts.Prune {
		options.Filters = append(options.Filters, "dangling=true")
	} else if !opts.All {
		options.Filters = append(options.Filters, "intermediate=false")
	}
	options.Force = opts.Force

	rmiReports, rmiErrors := engine.ImageRuntime().RemoveImages(context.Background(), opts.ImageNamesOrIDs, options)
	for _, r := range rmiReports {
		for _, u := range r.Untagged {
			fmt.Printf("untagged: %s\n", u)
		}
	}
	for _, r := range rmiReports {
		if r.Removed {
			fmt.Printf("%s\n", r.ID)
		}
	}

	var multiE *multierror.Error
	multiE = multierror.Append(multiE, rmiErrors...)
	return multiE.ErrorOrNil()
}
