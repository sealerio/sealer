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

	"github.com/containers/buildah/define"
	"github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/json"

	"github.com/sealerio/sealer/build/buildimage"
	"github.com/sealerio/sealer/build/kubefile/parser"
	"github.com/sealerio/sealer/common"
	version2 "github.com/sealerio/sealer/pkg/define/application/version"
	v12 "github.com/sealerio/sealer/pkg/define/image/v1"
	bc "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/image/save"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/imageengine/buildah"
	"github.com/sealerio/sealer/pkg/rootfs"
	v1 "github.com/sealerio/sealer/types/api/v1"
	osi "github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/yaml"
	"github.com/sealerio/sealer/version"
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
build with args:
  sealer build -f Kubefile -t my-kubernetes:1.19.8 --build-arg MY_ARG=abc,PASSWORD=Sealer123 .
build with image type:
  sealer build -f Kubefile -t my-kubernetes:1.19.8 --type=app-installer .
  sealer build -f Kubefile -t my-kubernetes:1.19.8 --type=kube-installer(default) .
  app-installer type image will not install kubernetes.
build multi-platform image:
	sealer build -f Kubefile -t my-kubernetes:1.19.8 --platform linux/amd64,linux/arm64
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
			if len(buildFlags.Tag) == 0 {
				return errors.New("--tag should be specified")
			}

			if len(args) > 0 {
				buildFlags.ContextDir = args[0]
			}
			return buildSealerImage()
		},
	}
	buildCmd.Flags().StringVarP(&buildFlags.Kubefile, "file", "f", "Kubefile", "Kubefile filepath")
	buildCmd.Flags().StringVarP(&buildFlags.Tag, "tag", "t", "", "specify a name for ClusterImage")
	//todo we can support imageList Flag to download extra container image rather than copy it to rootfs
	buildCmd.Flags().StringVar(&buildFlags.ImageList, "image-list", "filepath", "`pathname` of imageList filepath, if set, sealer will read its content and download extra container")
	buildCmd.Flags().StringVar(&buildFlags.ImageListWithAuth, "image-list-with-auth", "", "`pathname` of imageListWithAuth.yaml filepath, if set, sealer will read its content and download extra container images to rootfs(not usually used)")
	buildCmd.Flags().StringVar(&buildFlags.PullPolicy, "pull", "ifnewer", "pull policy. Allow for --pull, --pull=true, --pull=false, --pull=never, --pull=always, --pull=ifnewer")
	buildCmd.Flags().StringVar(&buildFlags.ImageType, "type", v12.KubeInstaller, fmt.Sprintf("specify the image type, --type=%s, --type=%s, default is %s", v12.KubeInstaller, v12.AppInstaller, v12.KubeInstaller))
	buildCmd.Flags().StringSliceVar(&buildFlags.Platforms, "platform", []string{parse.DefaultPlatform()}, "set the target platform, --platform=linux/amd64 or --platform=linux/amd64/v7. Multi-platform will be like --platform=linux/amd64,linux/amd64/v7")
	buildCmd.Flags().StringSliceVar(&buildFlags.BuildArgs, "build-arg", []string{}, "set custom build args")
	buildCmd.Flags().StringSliceVar(&buildFlags.Annotations, "annotation", []string{}, "add annotations for image. Format like --annotation key=[value]")
	buildCmd.Flags().StringSliceVar(&buildFlags.Labels, "label", []string{getSealerLabel()}, "add labels for image. Format like --label key=[value]")
	buildCmd.Flags().BoolVar(&buildFlags.NoCache, "no-cache", false, "do not use existing cached images for building. Build from the start with a new set of cached layers.")

	supportedImageType := map[string]struct{}{v12.KubeInstaller: {}, v12.AppInstaller: {}}
	if _, ok := supportedImageType[buildFlags.ImageType]; !ok {
		logrus.Fatalf("image type %s is not supported", buildFlags.ImageType)
	}

	return buildCmd
}

func buildSealerImage() error {
	engine, err := imageengine.NewImageEngine(bc.EngineGlobalConfigurations{})
	if err != nil {
		return errors.Wrap(err, "failed to initiate image engine")
	}

	kubefileParser := parser.NewParser(rootfs.GlobalManager.App().Root(), buildFlags, engine)
	result, err := getKubefileParseResult(buildFlags.ContextDir, buildFlags.Kubefile, kubefileParser)
	if err != nil {
		return err
	}
	logrus.Debugf("the result of kubefile parse as follows:\n %+v \n", result)
	defer func() {
		if err2 := result.CleanLegacyContext(); err2 != nil {
			logrus.Warnf("error in cleaning legacy in build sealer image: %v", err2)
		}
	}()
	// save the parsed dockerfile to a temporary file
	// and give it to buildFlags(buildFlags.Kubefile = dockerfilePath)
	dockerfilePath, err := saveDockerfileAsTempfile(result.Dockerfile)
	if err != nil {
		return errors.Wrap(err, "failed to save docker file as tempfile")
	}
	defer func() {
		_ = os.Remove(dockerfilePath)
	}()

	// set the image extension to oci image annotation
	imageExtension := buildImageExtensionOnResult(result, buildFlags.ImageType)
	iejson, err := json.Marshal(imageExtension)
	if err != nil {
		return errors.Wrap(err, "failed to marshal image extension")
	}
	var (
		repoTag   = buildFlags.Tag
		randomStr = getRandomString(8)
		// use temp tag to do temp image build, because after build,
		// we need to download some container data loaded from rootfs to it.
		tempTag = repoTag + randomStr
	)

	isMultiPlatform := len(buildFlags.Platforms) > 1
	if isMultiPlatform {
		buildFlags.Manifest = tempTag
		buildFlags.Tag = ""
	} else {
		buildFlags.Tag = tempTag
	}

	// add annotations to image. Store some sealer specific information
	buildFlags.Kubefile = dockerfilePath
	buildFlags.Annotations = append(buildFlags.Annotations, fmt.Sprintf("%s=%s", v12.SealerImageExtension, string(iejson)))
	iid, err := engine.Build(&buildFlags)
	if err != nil {
		return errors.Errorf("error in building image, %v", err)
	}

	defer func() {
		for _, m := range []string{tempTag} {
			// the above image is intermediate image, we need to remove it when the build ends.
			if err := engine.RemoveImage(&bc.RemoveImageOptions{
				ImageNamesOrIDs: []string{m},
				Force:           true,
			}); err != nil {
				logrus.Debugf("failed to remove image %s, you need to remove it manually: %v", m, err)
			}
		}
	}()

	if isMultiPlatform {
		return buildWithMultiPlatform(engine, tempTag, repoTag)
	}

	return buildWithSinglePlatform(engine, iid, repoTag)
}

func buildWithMultiPlatform(engine imageengine.Interface, tempTag, manifest string) error {
	platformedImages, err := getImageWithMultiPlatform(tempTag, engine)
	if err != nil {
		return errors.Wrapf(err, "failed to get image list")
	}

	platformContainerIDMap, containerImageListMap, err := applyImagesWithMultiPlatform(engine, platformedImages)
	if err != nil {
		return err
	}
	for containerID, containerImageList := range containerImageListMap {
		if err := updateContainerImageListAnnotations(engine, containerID, containerImageList); err != nil {
			return err
		}
	}

	for _, pi := range platformedImages {
		containerID := platformContainerIDMap[pi.platform.ToString()]
		if err := commitContainer(engine, &commitContainerOpts{
			containerID: containerID,
			tag:         "",
			manifest:    manifest,
			platform:    pi.platform,
		}); err != nil {
			return err
		}
	}
	return nil
}

func buildWithSinglePlatform(engine imageengine.Interface, imageID, tag string) error {
	_os, arch, variant, err := parse.Platform(buildFlags.Platforms[0])
	if err != nil {
		return errors.Wrap(err, "failed to parse platform")
	}
	platform := &v1.Platform{OS: _os, Architecture: arch, Variant: variant}

	containerID, containerImageList, err := applyImagesWithSinglePlatform(engine, imageID, platform)
	if err != nil {
		return err
	}

	if err := updateContainerImageListAnnotations(engine, containerID, containerImageList); err != nil {
		return err
	}

	return commitContainer(engine, &commitContainerOpts{
		containerID: containerID,
		tag:         tag,
		manifest:    "",
		platform:    *platform,
	})
}

type platformedImage struct {
	platform      v1.Platform
	imageNameOrID string
}

func getImageWithMultiPlatform(tempTag string, engine imageengine.Interface) ([]platformedImage, error) {
	var platformedImages []platformedImage
	manifestList, err := engine.LookupManifest(tempTag)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup manifest")
	}

	for _, p := range buildFlags.Platforms {
		_os, arch, variant, err := parse.Platform(p)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse platform")
		}

		img, err := manifestList.LookupInstance(context.TODO(), arch, _os, variant)
		if err != nil {
			return nil, err
		}

		platformedImages = append(platformedImages,
			platformedImage{imageNameOrID: img.ID(),
				platform: v1.Platform{OS: _os, Architecture: arch, Variant: variant}})
	}
	return platformedImages, nil
}

func applyImagesWithMultiPlatform(engine imageengine.Interface, platformedImages []platformedImage) (map[string]string, map[string][]*v12.ContainerImage, error) {
	var platformContainerIDMap = make(map[string]string)
	var containerImageList = make(map[string][]*v12.ContainerImage)
	for _, pi := range platformedImages {
		containerID, tmpContainerImageList, err := applyRegistryToImage(engine, pi.imageNameOrID, pi.platform)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error in apply registry data into image")
		}
		containerImageList[containerID] = tmpContainerImageList
		platformContainerIDMap[pi.platform.ToString()] = containerID
	}

	return platformContainerIDMap, containerImageList, nil
}

func applyImagesWithSinglePlatform(engine imageengine.Interface, imageID string, platform *v1.Platform) (string, []*v12.ContainerImage, error) {
	containerID, containerImageList, err := applyRegistryToImage(engine, imageID, *platform)
	if err != nil {
		return "", nil, errors.Wrap(err, "error in apply registry data into image")
	}

	return containerID, containerImageList, nil
}

func applyRegistryToImage(engine imageengine.Interface, imageID string, platform v1.Platform) (string, []*v12.ContainerImage, error) {
	_os, arch, variant := platform.OS, platform.Architecture, platform.Variant
	// this temporary file is used to execute image pull, and save it to /registry.
	// engine.BuildRootfs will generate an image rootfs, and link the rootfs to temporary dir(temp sealer rootfs).
	tmpDir, err := os.MkdirTemp("", "sealer")
	if err != nil {
		return "", nil, err
	}

	defer func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			logrus.Warnf("failed to rm link dir to rootfs: %v : %v", tmpDir, err)
		}
	}()

	tmpDirForLink := filepath.Join(tmpDir, "tmp-rootfs")
	containerID, err := engine.CreateWorkingContainer(&bc.BuildRootfsOptions{
		ImageNameOrID: imageID,
		DestDir:       tmpDirForLink,
	})
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to create working container, imageid: %s", imageID)
	}

	var containerImageList []*v12.ContainerImage

	// download container image from `imageList`
	if buildFlags.ImageList != "" && osi.IsFileExist(buildFlags.ImageList) {
		images, err := osi.NewFileReader(buildFlags.ImageList).ReadLines()
		if err != nil {
			return "", nil, err
		}
		for _, image := range images {
			logrus.Debugf("get container image(%s) with platform(%s) from build flag image list",
				image, platform.ToString())
			containerImageList = append(containerImageList, &v12.ContainerImage{
				Image:    image,
				AppName:  "",
				Platform: &platform,
			})
		}
		formatImages := buildimage.FormatImages(images)
		ctx := context.Background()
		imageSave := save.NewImageSaver(ctx)
		if err := imageSave.SaveImages(formatImages, filepath.Join(tmpDirForLink, common.RegistryDirName), v1.Platform{
			Architecture: arch,
			OS:           _os,
			Variant:      variant,
		}); err != nil {
			return "", nil, err
		}
	}

	// automatically parses container images and stores them
	parsedContainerImageList, err := buildimage.ParseContainerImageList(tmpDirForLink)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to parse container image list")
	}
	// add platform info for container image
	for _, containerImage := range parsedContainerImageList {
		logrus.Debugf("get container image(%s) with platform(%s) for app(%s) from build flag image list",
			containerImage.Image, containerImage.AppName, platform.ToString())
		containerImage.Platform = &platform
		containerImageList = append(containerImageList, containerImage)
	}
	registry := buildimage.NewRegistry(v1.Platform{
		Architecture: arch,
		OS:           _os,
		Variant:      variant,
	})
	if err := registry.SaveImages(tmpDirForLink, v12.GetImageSliceFromContainerImageList(parsedContainerImageList)); err != nil {
		return "", nil, errors.Wrap(err, "failed to download container images")
	}

	// download container image from `imageListWithAuth.yaml`
	if buildFlags.ImageListWithAuth != "" && osi.IsFileExist(buildFlags.ImageListWithAuth) {
		// pares middleware file: imageListWithAuth.yaml
		var imageSectionList []buildimage.ImageSection
		if err := yaml.UnmarshalFile(buildFlags.ImageListWithAuth, &imageSectionList); err != nil {
			return "", nil, err
		}
		for _, imageSection := range imageSectionList {
			for _, image := range imageSection.Images {
				logrus.Debugf("get container image(%s) with platform(%s) from build flag image list",
					image, platform.ToString())
				containerImageList = append(containerImageList, &v12.ContainerImage{
					Image:    image,
					AppName:  "",
					Platform: &platform,
				})
			}
		}
		if err := buildimage.NewMiddlewarePuller(v1.Platform{
			Architecture: arch,
			OS:           _os,
			Variant:      variant,
		}).PullWithImageSection(tmpDirForLink, imageSectionList); err != nil {
			return "", nil, err
		}
	}

	return containerID, containerImageList, nil
}

type commitContainerOpts struct {
	containerID string
	tag         string
	manifest    string
	platform    v1.Platform
}

func updateContainerImageListAnnotations(engine imageengine.Interface, containerID string, containerImageList []*v12.ContainerImage) error {
	logrus.Debugf("succcss to get containerImageList: %v", containerImageList)
	if len(containerImageList) == 0 {
		return nil
	}
	containerImageListJSON, err := json.Marshal(containerImageList)
	if err != nil {
		return errors.Wrap(err, "failed to marshal container image list")
	}
	containerImageListAnnotations := fmt.Sprintf("%s=%s",
		v12.SealerImageContainerImageList, string(containerImageListJSON))
	if err := engine.Config(&bc.ConfigOptions{
		ContainerID: containerID,
		Annotations: []string{containerImageListAnnotations},
	}); err != nil {
		return errors.Wrapf(err, "failed to config container images list")
	}
	return nil
}

func commitContainer(engine imageengine.Interface, opts *commitContainerOpts) error {
	tag := opts.tag
	manifest := opts.manifest
	containerID := opts.containerID
	platform := opts.platform
	id, err := engine.Commit(&bc.CommitOptions{
		Format:      cli.DefaultFormat(),
		Rm:          true,
		ContainerID: containerID,
		Image:       tag,
		Manifest:    manifest,
	})

	if err != nil {
		return errors.Wrapf(err, "failed to commit image tag: %s, manifest: %s", tag, manifest)
	}

	if len(manifest) > 0 {
		logrus.Infof("image(%s) committed to manifest %s, id: %s", platform.ToString(), manifest, id)
	} else {
		logrus.Infof("image(%s) named as %s, id: %s", platform.ToString(), tag, id)
	}

	return nil
}

func saveDockerfileAsTempfile(dockerFileContent string) (string, error) {
	f, err := os.CreateTemp("/tmp", "sealer-dockerfile")
	if err != nil {
		return "", err
	}

	defer func() {
		if err != nil {
			_ = os.Remove(f.Name())
		}
	}()

	_, err = f.WriteString(dockerFileContent)
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}

func buildImageExtensionOnResult(result *parser.KubefileResult, imageType string) *v12.ImageExtension {
	extension := &v12.ImageExtension{
		Type:          imageType,
		Applications:  []version2.VersionedApplication{},
		Launch:        v12.Launch{},
		SchemaVersion: v12.ImageSpecSchemaVersionV1Beta1,
		BuildClient: v12.BuildClient{
			SealerVersion:  version.Get().GitVersion,
			BuildahVersion: define.Version,
		},
	}

	for _, app := range result.Applications {
		extension.Applications = append(extension.Applications, app)
	}
	extension.Launch.Cmds = result.RawCmds
	extension.Launch.AppNames = result.AppNames
	extension.Launch.AppConfigs = result.ApplicationConfigs
	return extension
}

func getKubefileParseResult(contextDir, file string, kubefileParser *parser.KubefileParser) (*parser.KubefileResult, error) {
	kubefile, err := getKubefile(contextDir, file)
	if err != nil {
		return nil, err
	}

	kfr, err := os.Open(filepath.Clean(kubefile))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = kfr.Close()
	}()

	kr, err := kubefileParser.ParseKubefile(kfr)
	if err != nil {
		return nil, err
	}

	return kr, nil
}

func getKubefile(contextDir, file string) (string, error) {
	var (
		kubefile = file
		err      error
	)

	ctxDir, err := getContextDir(contextDir)
	if err != nil {
		return "", err
	}

	if len(kubefile) == 0 {
		kubefile, err = buildah.DiscoverKubefile(ctxDir)
		if err != nil {
			return "", err
		}
	}
	return kubefile, nil
}

func getContextDir(cxtDir string) (string, error) {
	var (
		contextDir = cxtDir
		err        error
	)
	if len(contextDir) == 0 {
		contextDir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	} else {
		// It was local.  Use it as is.
		contextDir, err = filepath.Abs(contextDir)
		if err != nil {
			return "", err
		}
	}

	return contextDir, nil
}

func getSealerLabel() string {
	return "io.sealer.version=" + version.Get().GitVersion
}

func getRandomString(n int) string {
	randBytes := make([]byte, n/2)
	_, _ = rand.Read(randBytes)
	return fmt.Sprintf("%x", randBytes)
}
