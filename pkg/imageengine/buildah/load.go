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

	"os"

	"strings"

	"github.com/containers/common/libimage"
)

func (engine *Engine) Load(opts *options.LoadOptions) error {
	// Download the input file if needed.
	//if strings.HasPrefix(opts.Input, "https://") || strings.HasPrefix(opts.Input, "http://") {
	//	tmpdir, err := util.DefaultContainerConfig().ImageCopyTmpDir()
	//	if err != nil {
	//		return err
	//	}
	//	tmpfile, err := download.FromURL(tmpdir, loadOpts.Input)
	//	if err != nil {
	//		return err
	//	}
	//	defer os.Remove(tmpfile)
	//	loadOpts.Input = tmpfile
	//}

	if _, err := os.Stat(opts.Input); err != nil {
		return err
	}

	loadOpts := &libimage.LoadOptions{}
	if !opts.Quiet {
		loadOpts.Writer = os.Stderr
	}

	loadedImages, err := engine.ImageRuntime().Load(context.Background(), opts.Input, loadOpts)
	if err != nil {
		return err
	}
	fmt.Println("Loaded image: " + strings.Join(loadedImages, "\nLoaded image: "))
	return nil
}
