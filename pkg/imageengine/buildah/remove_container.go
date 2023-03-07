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

	"github.com/containers/buildah"
	"github.com/containers/buildah/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/pkg/define/options"
)

func (engine *Engine) RemoveContainer(opts *options.RemoveContainerOptions) error {
	if len(opts.ContainerNamesOrIDs) == 0 && !opts.All {
		return fmt.Errorf("container name of id must be specified")
	}
	if len(opts.ContainerNamesOrIDs) > 0 && opts.All {
		return fmt.Errorf("all can't be true if the containers are specified")
	}

	var lastError error
	var delContainerErrStr = "error removing container"
	store := engine.ImageStore()
	if opts.All {
		builders, err := buildah.OpenAllBuilders(store)
		if err != nil {
			return errors.Wrapf(err, "error reading build containers")
		}

		for _, builder := range builders {
			id := builder.ContainerID
			if err = builder.Delete(); err != nil {
				lastError = util.WriteError(os.Stderr, errors.Wrapf(err, "%s %q", delContainerErrStr, builder.Container), lastError)
				continue
			}
			logrus.Debugf("%s", id)
		}
	} else {
		for _, name := range opts.ContainerNamesOrIDs {
			builder, err := OpenBuilder(getContext(), store, name)
			if err != nil {
				lastError = util.WriteError(os.Stderr, errors.Wrapf(err, "%s %q", delContainerErrStr, name), lastError)
				continue
			}
			id := builder.ContainerID
			if err = builder.Delete(); err != nil {
				lastError = util.WriteError(os.Stderr, errors.Wrapf(err, "%s %q", delContainerErrStr, name), lastError)
				continue
			}
			logrus.Debugf("%s", id)
		}
	}
	return lastError
}
