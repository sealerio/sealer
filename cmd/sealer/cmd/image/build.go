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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/pkg/errors"
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

build with args:
  sealer build -f Kubefile -t my-kubernetes:1.19.8 --build-arg MY_ARG=abc,PASSWORD=Sealer123 .

build with image type:
  sealer build -f Kubefile -t my-kubernetes:1.19.8 --type=app-installer .
  sealer build -f Kubefile -t my-kubernetes:1.19.8 --type=kube-installer(default) .
  app-installer type image will not install kubernetes.
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
	//todo we can support imageList Flag to download extra container image rather than copy it to rootfs
	buildCmd.Flags().StringVar(&buildFlags.ImageList, "image-list", "filepath", "`pathname` of imageList filepath, if set, sealer will read its content and download extra container")
	buildCmd.Flags().StringVar(&buildFlags.ImageListWithAuth, "image-list-with-auth", "", "`pathname` of imageListWithAuth.yaml filepath, if set, sealer will read its content and download extra container images to rootfs(not usually used)")
	buildCmd.Flags().StringVar(&buildFlags.Platform, "platform", parse.DefaultPlatform(), "set the target platform, like linux/amd64 or linux/amd64/v7")
	buildCmd.Flags().StringVar(&buildFlags.PullPolicy, "pull", "ifnewer", "pull policy. Allow for --pull, --pull=true, --pull=false, --pull=never, --pull=always, --pull=ifnewer")
	buildCmd.Flags().BoolVar(&buildFlags.NoCache, "no-cache", false, "do not use existing cached images for building. Build from the start with a new set of cached layers.")
	buildCmd.Flags().StringVar(&buildFlags.ImageType, "type", v12.KubeInstaller, fmt.Sprintf("specify the image type, --type=%s, --type=%s, default is %s", v12.KubeInstaller, v12.AppInstaller, v12.KubeInstaller))
	buildCmd.Flags().StringSliceVarP(&buildFlags.Tags, "tag", "t", []string{}, "specify a name for ClusterImage")
	buildCmd.Flags().StringSliceVar(&buildFlags.BuildArgs, "build-arg", []string{}, "set custom build args")
	buildCmd.Flags().StringSliceVar(&buildFlags.Annotations, "annotation", []string{}, "add annotations for image. Format like --annotation key=[value]")
	buildCmd.Flags().StringSliceVar(&buildFlags.Labels, "label", []string{getSealerLabel()}, "add labels for image. Format like --label key=[value]")

	requiredFlags := []string{"tag"}
	for _, flag := range requiredFlags {
		if err := buildCmd.MarkFlagRequired(flag); err != nil {
			logrus.Fatal(err)
		}
	}

	supportedImageType := map[string]struct{}{v12.KubeInstaller: {}, v12.AppInstaller: {}}
	if _, ok := supportedImageType[buildFlags.ImageType]; !ok {
		logrus.Fatalf("image type %s is not supported", buildFlags.ImageType)
	}

	return buildCmd
}

func buildSealerImage() error {
	_os, arch, variant, err := parse.Platform(buildFlags.Platform)
	if err != nil {
		return err
	}

	engine, err := imageengine.NewImageEngine(bc.EngineGlobalConfigurations{})
	if err != nil {
		return errors.Wrap(err, "failed to initiate a builder")
	}

	kubefileParser := parser.NewParser(rootfs.GlobalManager.App().Root(), buildFlags, engine)
	result, err := getKubefileParseResult(buildFlags.ContextDir, buildFlags.Kubefile, kubefileParser)
	if err != nil {
		return err
	}
	logrus.Debugf("the result of kubefile parse as follows:\n %+v \n", &result)
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

	buildFlags.Kubefile = dockerfilePath
	buildFlags.Annotations = append(buildFlags.Annotations, fmt.Sprintf("%s=%s", v12.SealerImageExtension, string(iejson)))
	iid, err := engine.Build(&buildFlags)
	if err != nil {
		return errors.Errorf("error in building image, %v", err)
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

	// this temporary file is used to execute image pull, and save it to /registry.
	// engine.BuildRootfs will generate an image rootfs, and link the rootfs to temporary dir(temp sealer rootfs).
	tmpDir, err := os.MkdirTemp("", "sealer")
	if err != nil {
		return err
	}

	defer func() {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			logrus.Warnf("failed to rm link dir to rootfs: %v : %v", tmpDir, err)
		}
	}()

	tmpDirForLink := filepath.Join(tmpDir, "tmp-rootfs")
	cid, err := engine.CreateWorkingContainer(&bc.BuildRootfsOptions{
		ImageNameOrID: iid,
		DestDir:       tmpDirForLink,
	})
	if err != nil {
		return err
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

	if err := buildimage.NewRegistryDiffer(v1.Platform{
		Architecture: arch,
		OS:           _os,
		Variant:      variant,
	}).Process(tmpDirForLink, tmpDirForLink); err != nil {
		return err
	}

	// download container image form `imageListWithAuth.yaml`
	if err := buildimage.NewMiddlewarePuller(v1.Platform{
		Architecture: arch,
		OS:           _os,
		Variant:      variant,
	}).Pull(buildFlags.ImageListWithAuth, tmpDirForLink); err != nil {
		return err
	}

	if err := engine.Commit(&bc.CommitOptions{
		Format:      cli.DefaultFormat(),
		Rm:          true,
		ContainerID: cid,
		Image:       buildFlags.Tags[0],
	}); err != nil {
		return err
	}

	return nil
}

func saveDockerfileAsTempfile(dockerFileContent string) (string, error) {
	f, err := ioutil.TempFile("/tmp", "sealer-dockerfile")
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
		Type:         imageType,
		Applications: []version2.VersionedApplication{},
		Launch:       v12.Launch{},
	}

	for _, app := range result.Applications {
		extension.Applications = append(extension.Applications, app)
	}
	extension.Launch.Cmds = result.LaunchList
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
