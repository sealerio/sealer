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
	"io"
	"os"
	"path/filepath"

	"github.com/containers/common/libimage"
	"github.com/containers/common/libimage/manifests"
	"github.com/pkg/errors"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/utils/archive"
	osi "github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/os/fs"
	"github.com/sirupsen/logrus"
)

// Save image as tar file, if image is multi-arch image, will save all its instances and manifest name as tar file.
func (engine *Engine) Save(opts *options.SaveOptions) error {
	imageNameOrID := opts.ImageNameOrID
	imageTar := opts.Output
	store := engine.ImageStore()

	if len(imageNameOrID) == 0 {
		return errors.New("image name or id must be specified")
	}
	if opts.Compress && (opts.Format != OCIManifestDir && opts.Format != V2s2ManifestDir) {
		return errors.New("--compress can only be set when --format is either 'oci-dir' or 'docker-dir'")
	}

	img, _, err := engine.ImageRuntime().LookupImage(imageNameOrID, &libimage.LookupImageOptions{
		ManifestList: true,
	})
	if err != nil {
		return err
	}

	isManifest, err := img.IsManifestList(getContext())
	if err != nil {
		return err
	}

	if !isManifest {
		return engine.saveOneImage(imageNameOrID, opts.Format, imageTar, opts.Compress)
	}

	// save multi-arch images :including each platform images and manifest.
	var pathsToCompress []string

	if err := fs.FS.MkdirAll(filepath.Dir(imageTar)); err != nil {
		return fmt.Errorf("failed to create %s, err: %v", imageTar, err)
	}

	file, err := os.Create(filepath.Clean(imageTar))
	if err != nil {
		return fmt.Errorf("failed to create %s, err: %v", imageTar, err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			logrus.Errorf("failed to close file: %v", err)
		}
	}()

	tempDir, err := os.MkdirTemp(opts.TmpDir, "sealer-save-tmp")
	if err != nil {
		return fmt.Errorf("failed to create %s, err: %v", tempDir, err)
	}

	defer func() {
		err = os.RemoveAll(tempDir)
		if err != nil {
			logrus.Warnf("failed to delete %s: %v", tempDir, err)
		}
	}()

	// save each platform images
	imageName := img.Names()[0]
	logrus.Infof("image %q is a manifest list, looking up matching instance to save", imageNameOrID)
	manifestList, err := engine.ImageRuntime().LookupManifestList(imageName)
	if err != nil {
		return err
	}

	_, list, err := manifests.LoadFromImage(store, manifestList.ID())
	if err != nil {
		return err
	}

	for _, instanceDigest := range list.Instances() {
		images, err := store.ImagesByDigest(instanceDigest)
		if err != nil {
			return err
		}

		if len(images) == 0 {
			return fmt.Errorf("no image matched with digest %s", instanceDigest)
		}

		instance := images[0]
		instanceTar := filepath.Join(tempDir, instance.ID+".tar")

		// if instance has "Names", use the first one as saved name
		instanceName := instance.ID
		if len(instance.Names) > 0 {
			instanceName = instance.Names[0]
		}

		err = engine.saveOneImage(instanceName, opts.Format, instanceTar, opts.Compress)
		if err != nil {
			return err
		}

		pathsToCompress = append(pathsToCompress, instanceTar)
	}

	// save imageName to metadata file
	metaFile := filepath.Join(tempDir, common.DefaultMetadataName)
	if err = osi.NewAtomicWriter(metaFile).WriteFile([]byte(imageName)); err != nil {
		return fmt.Errorf("failed to write temp file %s, err: %v ", metaFile, err)
	}
	pathsToCompress = append(pathsToCompress, metaFile)

	// tar all materials
	tarReader, err := archive.TarWithRootDir(pathsToCompress...)
	if err != nil {
		return fmt.Errorf("failed to get tar reader for %s, err: %s", imageNameOrID, err)
	}
	defer func() {
		if err := tarReader.Close(); err != nil {
			logrus.Errorf("failed to close file: %v", err)
		}
	}()

	_, err = io.Copy(file, tarReader)

	return err
}

func (engine *Engine) saveOneImage(imageNameOrID, format, path string, compress bool) error {
	saveOptions := &libimage.SaveOptions{
		CopyOptions: libimage.CopyOptions{
			DirForceCompress:            compress,
			OciAcceptUncompressedLayers: false,
			// Force signature removal to preserve backwards compat.
			// See https://github.com/containers/podman/pull/11669#issuecomment-925250264
			RemoveSignatures: true,
		},
	}

	names := []string{imageNameOrID}
	return engine.ImageRuntime().Save(context.Background(), names, format, path, saveOptions)
}
