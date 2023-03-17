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
	"os"

	"github.com/containers/common/pkg/auth"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	"github.com/sealerio/sealer/pkg/define/options"
)

func (engine *Engine) Login(opts *options.LoginOptions) error {
	if len(opts.Domain) == 0 {
		return errors.Errorf("please specify a registry to login to")
	}

	systemCxt := engine.SystemContext()
	systemCxt.OCIInsecureSkipTLSVerify = opts.SkipTLSVerify
	systemCxt.DockerInsecureSkipTLSVerify = types.NewOptionalBool(opts.SkipTLSVerify)

	return auth.Login(context.TODO(),
		systemCxt,
		&auth.LoginOptions{
			AuthFile:           systemCxt.AuthFilePath,
			Password:           opts.Password,
			Username:           opts.Username,
			Stdout:             os.Stdout,
			AcceptRepositories: true,
		},
		[]string{opts.Domain})
}
