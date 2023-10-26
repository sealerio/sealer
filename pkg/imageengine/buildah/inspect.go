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
	"sort"
	"strings"

	"github.com/containers/buildah"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"

	"github.com/sealerio/sealer/build/kubefile/command"
	imagev1 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/define/options"
)

func (engine *Engine) Inspect(opts *options.InspectOptions) (*imagev1.ImageSpec, error) {
	if len(opts.ImageNameOrID) == 0 {
		return nil, errors.Errorf("image name or image id must be specified")
	}

	var (
		builder *buildah.Builder
		err     error
	)

	ctx := getContext()
	store := engine.ImageStore()
	newSystemCxt := engine.SystemContext()
	name := opts.ImageNameOrID

	builder, err = openImage(ctx, newSystemCxt, store, name)
	if err != nil {
		return nil, err
	}

	builderInfo := buildah.GetBuildInfo(builder)
	var manifest = ociv1.Manifest{}
	if err := json.Unmarshal([]byte(builderInfo.Manifest), &manifest); err != nil {
		return nil, errors.Wrapf(err, "failed to get manifest")
	}

	if len(manifest.Annotations) != 0 {
		delete(manifest.Annotations, imagev1.SealerImageExtension)
		delete(manifest.Annotations, imagev1.SealerImageContainerImageList)
	}

	imageExtension, err := getImageExtensionFromAnnotations(builderInfo.ImageAnnotations)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %s in image %s", imagev1.SealerImageExtension, opts.ImageNameOrID)
	}

	imageExtension.Labels = handleImageLabelOutput(builderInfo.OCIv1.Config.Labels)

	// NOTE: avoid duplicate content output
	builderInfo.OCIv1.Config.Labels = nil

	containerImageList, err := getContainerImagesFromAnnotations(builderInfo.ImageAnnotations)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %s in image %s", imagev1.SealerImageContainerImageList, opts.ImageNameOrID)
	}

	result := &imagev1.ImageSpec{
		ID:                 builderInfo.FromImageID,
		Name:               builderInfo.FromImage,
		Digest:             builderInfo.FromImageDigest,
		ManifestV1:         manifest,
		OCIv1:              builderInfo.OCIv1,
		ImageExtension:     imageExtension,
		ContainerImageList: containerImageList,
	}

	return result, nil
}

func getImageExtensionFromAnnotations(annotations map[string]string) (imagev1.ImageExtension, error) {
	extension := imagev1.ImageExtension{}
	extensionStr := annotations[imagev1.SealerImageExtension]
	if len(extensionStr) == 0 {
		return extension, fmt.Errorf("%s does not exist", imagev1.SealerImageExtension)
	}

	if err := json.Unmarshal([]byte(extensionStr), &extension); err != nil {
		return extension, errors.Wrapf(err, "failed to unmarshal %v", imagev1.SealerImageExtension)
	}
	return extension, nil
}

func getContainerImagesFromAnnotations(annotations map[string]string) ([]*imagev1.ContainerImage, error) {
	var containerImageList []*imagev1.ContainerImage
	annotationStr := annotations[imagev1.SealerImageContainerImageList]
	if len(annotationStr) == 0 {
		return nil, nil
	}

	if err := json.Unmarshal([]byte(annotationStr), &containerImageList); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal %v", imagev1.SealerImageContainerImageList)
	}
	return containerImageList, nil
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
		sort.Strings(supportedCNI)
		supportedCNIJSON, _ := json.Marshal(supportedCNI)
		result[command.LabelSupportedKubeCNIAlpha] = string(supportedCNIJSON)
	}
	if len(supportedCSI) != 0 {
		sort.Strings(supportedCSI)
		supportedCSIJSON, _ := json.Marshal(supportedCSI)
		result[command.LabelSupportedKubeCSIAlpha] = string(supportedCSIJSON)
	}

	return result
}
