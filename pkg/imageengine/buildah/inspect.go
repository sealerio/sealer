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
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/sealerio/sealer/build/kubefile/command"
	image_v1 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/define/options"

	"github.com/containers/buildah"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"golang.org/x/term"
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

	builderInfo := buildah.GetBuildInfo(builder)
	var manifest = ociv1.Manifest{}
	if err := json.Unmarshal([]byte(builderInfo.Manifest), &manifest); err != nil {
		return errors.Wrapf(err, "failed to get manifest")
	}
	imageExtension, err := GetImageExtensionFromAnnotations(builderInfo.ImageAnnotations)
	if err != nil {
		return errors.Wrapf(err, "failed to get %s in image %s", image_v1.SealerImageExtension, opts.ImageNameOrID)
	}
	imageExtension.Labels = handleImageLabelOutput(builderInfo.OCIv1.Config.Labels)
	// NOTE: avoid duplicate content output
	builderInfo.OCIv1.Config.Labels = nil

	containerImageList, err := GetContainerImagesFromAnnotations(builderInfo.ImageAnnotations)
	if err != nil {
		return errors.Wrapf(err, "failed to get %s in image %s", image_v1.SealerImageContainerImageList, opts.ImageNameOrID)
	}

	result := &image_v1.ImageSpec{
		ID:                 builderInfo.FromImageID,
		Name:               builderInfo.FromImage,
		Digest:             builderInfo.FromImageDigest,
		ManifestV1:         manifest,
		OCIv1:              builderInfo.OCIv1,
		ImageExtension:     imageExtension,
		ContainerImageList: containerImageList,
	}
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
		if err = t.Execute(os.Stdout, result); err != nil {
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
	return enc.Encode(result)
}

func handleImageLabelOutput(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return labels
	}

	var result = make(map[string]string)
	var supportedCNI []string
	var supportedCSI []string
	for k, v := range labels {
		if strings.HasPrefix(k, command.LabelKubeCNIPrefix) {
			supportedCNI = append(supportedCNI, strings.TrimPrefix(k, command.LabelKubeCNIPrefix))
			continue
		}
		if strings.HasPrefix(k, command.LabelKubeCSIPrefix) {
			supportedCSI = append(supportedCSI, strings.TrimPrefix(k, command.LabelKubeCSIPrefix))
			continue
		}
		result[k] = v
	}

	if len(supportedCNI) != 0 {
		supportedCNIJSON, _ := json.Marshal(supportedCNI)
		result[command.LabelSupportedKubeCNIAlpha] = string(supportedCNIJSON)
	}
	if len(supportedCSI) != 0 {
		supportedCSIJSON, _ := json.Marshal(supportedCSI)
		result[command.LabelSupportedKubeCSIAlpha] = string(supportedCSIJSON)
	}

	return result
}
