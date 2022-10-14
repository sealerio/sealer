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
	"github.com/containers/common/pkg/auth"

	"github.com/sealerio/sealer/pkg/define/options"

	"os"

	"github.com/pkg/errors"
)

func (engine *Engine) Logout(opts *options.LogoutOptions) error {
	if len(opts.Domain) == 0 {
		return errors.Errorf("registry must be given")
	}

	systemCxt := engine.SystemContext()
	return auth.Logout(systemCxt, &auth.LogoutOptions{
		AuthFile:           systemCxt.AuthFilePath,
		All:                opts.All,
		AcceptRepositories: true,
		Stdout:             os.Stdout,
	}, []string{opts.Domain})
}
