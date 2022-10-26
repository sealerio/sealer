// Copyright © 2021 Alibaba Group Holding Ltd.
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

package image

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	bc "github.com/sealerio/sealer/pkg/define/options"

	"github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/pkg/errors"
	"github.com/sealerio/sealer/build/buildimage"
	"github.com/sealerio/sealer/pkg/imageengine"
	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sealerio/sealer/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/json"
)

var buildFlags = bc.BuildOptions{}

var longNewBuildCmdDescription = `build command is used to generate a ClusterImage from specified Kubefile.
It organizes the specified Kubefile and input building context, and builds
a brand new ClusterImage.`

var exampleNewBuildCmd = `the current path is the context path, default build type is lite and use build cache

build:
	sealer build -f Kubefile -t my-kubernetes:1.19.8 .

build without cache:
	sealer build -f Kubefile -t my-kubernetes:1.19.8 --no-cache .

build without base:
	sealer build -f Kubefile -t my-kubernetes:1.19.8 --base=false .

build with args:
	sealer build -f Kubefile -t my-kubernetes:1.19.8 --build-arg MY_ARG=abc,PASSWORD=Sealer123 .
`

// NewBuildCmd buildCmd represents the build command
func NewBuildCmd() *cobra.Command {
	buildCmd := &cobra.Command{
		Use:     "build [flags] PATH",
		Short:   "build a ClusterImage from a Kubefile",
		Long:    longNewBuildCmdDescription,
		Args:    cobra.MaximumNArgs(1),
		Example: exampleNewBuildCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				buildFlags.ContextDir = args[0]
			}
			return buildSealerImage()
		},
	}
	buildCmd.Flags().StringVarP(&buildFlags.Kubefile, "file", "f", "Kubefile", "Kubefile filepath")

	buildCmd.Flags().StringVar(&buildFlags.PullPolicy, "pull", "", "pull policy. Allow for --pull, --pull=true, --pull=false, --pull=never, --pull=always")
	buildCmd.Flags().BoolVar(&buildFlags.NoCache, "no-cache", false, "do not use existing cached images for building. Build from the start with a new set of cached layers.")
	buildCmd.Flags().BoolVar(&buildFlags.Base, "base", true, "build with base image, default value is true.")
	buildCmd.Flags().StringSliceVar(&buildFlags.Platforms, "platform", []string{parse.DefaultPlatform()}, "set the target platform, --platform=linux/amd64 or --platform=linux/amd64/v7. Multi-platform will be like --platform=linux/amd64,linux/amd64/v7")
	buildCmd.Flags().StringVarP(&buildFlags.Tag, "tag", "t", "", "specify a name for ClusterImage")
	buildCmd.Flags().StringSliceVar(&buildFlags.BuildArgs, "build-arg", []string{}, "set custom build args")
	buildCmd.Flags().StringSliceVar(&buildFlags.Annotations, "annotation", []string{}, "add annotations for image. Format like --annotation key=[value]")
	buildCmd.Flags().StringSliceVar(&buildFlags.Labels, "label", []string{getSealerLabel()}, "add labels for image. Format like --label key=[value]")
	requiredFlags := []string{"tag"}
	for _, flag := range requiredFlags {
		if err := buildCmd.MarkFlagRequired(flag); err != nil {
			logrus.Fatal(err)
		}
	}

	return buildCmd
}

func buildSealerImage() error {
	finalTagOrManifest := buildFlags.Tag
	tempTagOrManifest := buildFlags.Tag + getRandomString(8)
	buildFlags.Tag = tempTagOrManifest

	engine, err := imageengine.NewImageEngine(bc.EngineGlobalConfigurations{})
	if err != nil {
		return errors.Wrap(err, "failed to initiate image engine")
	}

	extension := v1.ImageExtension{}
	extensionBytes, err := json.Marshal(extension)
	if err != nil {
		return err
	}

	// add images to manifest
	// this may lead to create a manifest, or just add them to the existing manifest
	isMultiPlatform := len(buildFlags.Platforms) > 1
	if isMultiPlatform {
		buildFlags.Manifest = buildFlags.Tag
		buildFlags.Tag = ""
	}

	// add annotations to image. Store some sealer specific information
	buildFlags.Annotations = append(buildFlags.Annotations, fmt.Sprintf("%s=%s", v1.SealerImageExtension, string(extensionBytes)))
	iid, err := engine.Build(&buildFlags)
	if err != nil {
		return errors.Errorf("error in building image, %v", err)
	}

	type platformedImage struct {
		platform      v1.Platform
		imageNameOrID string
	}
	platformedImages := []platformedImage{}

	//ImageNamesOrIDs := []string{}
	if !isMultiPlatform {
		//ImageNamesOrIDs = append(ImageNamesOrIDs, iid)
		_os, arch, variant, err := parse.Platform(buildFlags.Platforms[0])
		if err != nil {
			return errors.Wrap(err, "failed to parse platform")
		}

		platformedImages = append(platformedImages,
			platformedImage{imageNameOrID: iid,
				platform: v1.Platform{OS: _os, Architecture: arch, Variant: variant}})
	}

	defer func() {
		// the above image is intermediate image, we need to remove it when the build ends.
		if err := engine.RemoveImage(&bc.RemoveImageOptions{
			ImageNamesOrIDs: []string{iid},
			Force:           true,
		}); err != nil {
			logrus.Warnf("failed to remove image %s, you need to remove it manually: %v", iid, err)
		}
	}()

	if isMultiPlatform {
		manifestList, err := engine.LookupManifest(buildFlags.Manifest)
		if err != nil {
			return errors.Wrap(err, "failed to lookup manifest")
		}
		manifestList.Inspect()

		for _, p := range buildFlags.Platforms {
			_os, arch, variant, err := parse.Platform(p)
			if err != nil {
				return errors.Wrap(err, "failed to parse platform")
			}

			img, err := manifestList.LookupInstance(context.TODO(), arch, _os, variant)
			if err != nil {
				return err
			}

			platformedImages = append(platformedImages,
				platformedImage{imageNameOrID: img.ID(),
					platform: v1.Platform{OS: _os, Architecture: arch, Variant: variant}})
		}
	}

	for _, pi := range platformedImages {
		if err := applyRegistryToImage(pi.imageNameOrID, finalTagOrManifest, isMultiPlatform, pi.platform, engine); err != nil {
			return errors.Wrap(err, "error in apply registry data into image")
		}
	}

	return nil
}

func applyRegistryToImage(imageID, tagOrManifest string, isMultiPlatform bool, platform v1.Platform, engine imageengine.Interface) error {
	_os, arch, variant := platform.OS, platform.Architecture, platform.Variant

	// this temporary file is used to execute image pull, and save it to /registry.
	// engine.BuildRootfs will generate an image rootfs, and link the rootfs to temporary dir(temp sealer rootfs).
	tmpDir, err := os.MkdirTemp("", "sealer")
	if err != nil {
		return err
	}

	tmpDirForLink := filepath.Join(tmpDir, "tmp-rootfs")
	cid, err := engine.BuildRootfs(&bc.BuildRootfsOptions{
		ImageNameOrID: imageID,
		DestDir:       tmpDirForLink,
	})
	if err != nil {
		return err
	}

	defer func() {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			logrus.Warnf("failed to rm link dir to rootfs: %v : %v", tmpDir, err)
		}
	}()

	differ := buildimage.NewRegistryDiffer(v1.Platform{
		Architecture: arch,
		OS:           _os,
		Variant:      variant,
	})

	if err = differ.Process(tmpDirForLink, tmpDirForLink); err != nil {
		return errors.Wrap(err, "failed to download container images")
	}

	tag := ""
	manifest := ""
	if isMultiPlatform {
		manifest = tagOrManifest
	} else {
		tag = tagOrManifest
	}

	if err = engine.Commit(&bc.CommitOptions{
		Format:      cli.DefaultFormat(),
		Rm:          true,
		ContainerID: cid,
		Image:       tag,
		Manifest:    manifest,
	}); err != nil {
		return errors.Wrap(err, "failed to commit registry into image")
	}

	return nil
}

func getSealerLabel() string {
	return "io.sealer.version=" + version.Get().GitVersion
}

func getRandomString(n int) string {
	randBytes := make([]byte, n/2)
	rand.Read(randBytes)
	return fmt.Sprintf("%x", randBytes)
}
