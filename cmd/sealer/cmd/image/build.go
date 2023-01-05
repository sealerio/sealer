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
	buildCmd.Flags().BoolVarP(&buildFlags.DownloadContainerImage, "download-container-image", "d", true, "save the container image generated during the build process.")

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
		if !buildFlags.DownloadContainerImage {
			for _, m := range []string{tempTag} {
				// the above image is intermediate image, we need to remove it when the build ends.
				if err := engine.RemoveImage(&bc.RemoveImageOptions{
					ImageNamesOrIDs: []string{m},
					Force:           true,
				}); err != nil {
					logrus.Debugf("failed to remove image %s, you need to remove it manually: %v", m, err)
				}
			}
		}
	}()

	if isMultiPlatform {
		return commitMultiPlatformImage(tempTag, repoTag, engine)
	}

	return commitSingleImage(iid, repoTag, engine)
}

type platformedImage struct {
	platform      v1.Platform
	imageNameOrID string
}

func commitMultiPlatformImage(tempTag, manifest string, engine imageengine.Interface) error {
	var platformedImages []platformedImage
	manifestList, err := engine.LookupManifest(tempTag)
	if err != nil {
		return errors.Wrap(err, "failed to lookup manifest")
	}

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

	for _, pi := range platformedImages {
		if err := applyRegistryToImage(pi.imageNameOrID, "", manifest, pi.platform, engine); err != nil {
			return errors.Wrap(err, "error in apply registry data into image")
		}
	}

	return nil
}

func commitSingleImage(iid string, tag string, engine imageengine.Interface) error {
	_os, arch, variant, err := parse.Platform(buildFlags.Platforms[0])
	if err != nil {
		return errors.Wrap(err, "failed to parse platform")
	}

	if err := applyRegistryToImage(iid, tag, "", v1.Platform{OS: _os, Architecture: arch, Variant: variant}, engine); err != nil {
		return errors.Wrap(err, "error in apply registry data into image")
	}

	return nil
}

func applyRegistryToImage(imageID, tag, manifest string, platform v1.Platform, engine imageengine.Interface) error {
	if !buildFlags.DownloadContainerImage {
		return nil
	}

	_os, arch, variant := platform.OS, platform.Architecture, platform.Variant
	// this temporary file is used to execute image pull, and save it to /registry.
	// engine.BuildRootfs will generate an image rootfs, and link the rootfs to temporary dir(temp sealer rootfs).
	tmpDir, err := os.MkdirTemp("", "sealer")
	if err != nil {
		return err
	}

	defer func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			logrus.Warnf("failed to rm link dir to rootfs: %v : %v", tmpDir, err)
		}
	}()

	tmpDirForLink := filepath.Join(tmpDir, "tmp-rootfs")
	cid, err := engine.CreateWorkingContainer(&bc.BuildRootfsOptions{
		ImageNameOrID: imageID,
		DestDir:       tmpDirForLink,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create working container, imageid: %s", imageID)
	}

	// download container image form `imageList`
	if buildFlags.ImageList != "" && osi.IsFileExist(buildFlags.ImageList) {
		images, err := osi.NewFileReader(buildFlags.ImageList).ReadLines()
		if err != nil {
			return err
		}
		formatImages := buildimage.FormatImages(images)
		ctx := context.Background()
		imageSave := save.NewImageSaver(ctx)
		if err := imageSave.SaveImages(formatImages, filepath.Join(tmpDirForLink, common.RegistryDirName), v1.Platform{
			Architecture: arch,
			OS:           _os,
			Variant:      variant,
		}); err != nil {
			return err
		}
	}

	differ := buildimage.NewRegistryDiffer(v1.Platform{
		Architecture: arch,
		OS:           _os,
		Variant:      variant,
	})

	// TODO optimize the differ.
	if err := differ.Process(tmpDirForLink, tmpDirForLink); err != nil {
		return errors.Wrap(err, "failed to download container images")
	}

	// download container image form `imageListWithAuth.yaml`
	if err := buildimage.NewMiddlewarePuller(v1.Platform{
		Architecture: arch,
		OS:           _os,
		Variant:      variant,
	}).Pull(buildFlags.ImageListWithAuth, tmpDirForLink); err != nil {
		return err
	}

	id, err := engine.Commit(&bc.CommitOptions{
		Format:      cli.DefaultFormat(),
		Rm:          true,
		ContainerID: cid,
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
