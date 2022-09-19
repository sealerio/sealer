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
	"encoding/json"
	"fmt"

	"github.com/sealerio/sealer/pkg/define/options"

	"github.com/containers/buildah"
	"github.com/pkg/errors"
	"golang.org/x/term"

	"os"
	"regexp"
	"text/template"
)

const (
	//inspectTypeContainer = "container"
	inspectTypeImage = "image"
	//inspectTypeManifest  = "manifest"
)

func (engine *Engine) Inspect(opts *options.InspectOptions) error {
	if len(opts.ImageNameOrID) == 0 {
		return errors.Errorf("image name or image id must be specified.")
	}
	var (
		builder *buildah.Builder
		err     error
	)

	ctx := getContext()
	store := engine.ImageStore()
	name := opts.ImageNameOrID

	switch opts.InspectType {
	case inspectTypeImage:
		builder, err = openImage(ctx, engine.SystemContext(), store, name)
		if err != nil {
			return err
		}
	//case inspectTypeManifest:
	default:
		return errors.Errorf("the only recognized type is %q", inspectTypeImage)
	}

	out := buildah.GetBuildInfo(builder)
	if opts.Format != "" {
		format := opts.Format
		if matched, err := regexp.MatchString("{{.*}}", format); err != nil {
			return errors.Wrapf(err, "error validating format provided: %s", format)
		} else if !matched {
			return errors.Errorf("error invalid format provided: %s", format)
		}
		t, err := template.New("format").Parse(format)
		if err != nil {
			return errors.Wrapf(err, "Template parsing error")
		}
		if err = t.Execute(os.Stdout, out); err != nil {
			return err
		}
		if term.IsTerminal(int(os.Stdout.Fd())) {
			fmt.Println()
		}
		return nil
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	if term.IsTerminal(int(os.Stdout.Fd())) {
		enc.SetEscapeHTML(false)
	}
	return enc.Encode(out)
}
