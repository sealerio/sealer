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
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/common/libimage"
	"github.com/go-errors/errors"
	"github.com/sirupsen/logrus"

	"github.com/containers/common/pkg/auth"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/utils/archive"
)

var LoadError = errors.Errorf("failed to load new image")

func (engine *Engine) Load(opts *options.LoadOptions) error {
	imageSrc := opts.Input
	if _, err := os.Stat(imageSrc); err != nil {
		return err
	}

	systemCxt := engine.SystemContext()
	if err := auth.CheckAuthFile(systemCxt.AuthFilePath); err != nil {
		return err
	}

	systemCxt.BigFilesTemporaryDir = opts.TmpDir

	loadOpts := &libimage.LoadOptions{}
	if !opts.Quiet {
		loadOpts.Writer = os.Stderr
	}

	srcFile, err := os.Open(filepath.Clean(imageSrc))
	if err != nil {
		return fmt.Errorf("failed to open %s, err : %v", imageSrc, err)
	}

	defer func() {
		if err := srcFile.Close(); err != nil {
			logrus.Errorf("failed to close file: %v", err)
		}
	}()

	tempDir, err := os.MkdirTemp(opts.TmpDir, "sealer-load-tmp")
	if err != nil {
		return fmt.Errorf("failed to create %s, err: %v", tempDir, err)
	}

	defer func() {
		err = os.RemoveAll(tempDir)
		if err != nil {
			logrus.Errorf("failed to delete %s: %v", tempDir, err)
		}
	}()

	// decompress tar file
	if _, err = archive.Decompress(srcFile, tempDir, archive.Options{Compress: false}); err != nil {
		return err
	}

	metaFile := filepath.Join(tempDir, common.DefaultMetadataName)

	// assume it is single image to load
	if _, err := os.Stat(metaFile); err != nil {
		return engine.loadOneImage(imageSrc, loadOpts)
	}

	// get manifestName
	metaBytes, err := ioutil.ReadFile(filepath.Clean(metaFile))
	if err != nil {
		return err
	}

	manifestName := string(metaBytes)
	// delete it if manifestName is already used
	if _, err := engine.ImageRuntime().LookupManifestList(manifestName); err == nil {
		logrus.Warnf("%s is already in use, will delete it", manifestName)
		if delErr := engine.DeleteManifests([]string{manifestName}, &options.ManifestDeleteOpts{}); delErr != nil {
			return fmt.Errorf("%s is already in use: %v", manifestName, delErr)
		}
	}

	// walk through temp dir to load each instance
	var instancesIDs []string
	err = filepath.Walk(tempDir, func(path string, f fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(f.Name(), ".tar") {
			return nil
		}

		instanceSrc := filepath.Join(tempDir, f.Name())
		err = engine.loadOneImage(instanceSrc, loadOpts)
		if err != nil {
			return fmt.Errorf("failed to load %s from %s: %v", f.Name(), imageSrc, err)
		}

		instancesIDs = append(instancesIDs, strings.TrimSuffix(f.Name(), ".tar"))
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to load image instance %v", err)
	}

	// create a new manifest and add instance to it.
	_, err = engine.CreateManifest(manifestName, &options.ManifestCreateOpts{})
	if err != nil {
		return fmt.Errorf("failed to create new manifest %s :%v ", manifestName, err)
	}

	defer func() {
		if errors.Is(err, LoadError) {
			err = engine.DeleteManifests([]string{manifestName}, &options.ManifestDeleteOpts{})
			if err != nil {
				logrus.Errorf("failed to delete manifest %s :%v ", manifestName, err)
			}
		}
	}()

	for _, imageID := range instancesIDs {
		err = engine.AddToManifest(manifestName, imageID, &options.ManifestAddOpts{})
		if err != nil {
			logrus.Errorf("failed to add new image %s to %s :%v ", imageID, manifestName, err)
			return LoadError
		}
	}

	return nil
}

func (engine *Engine) loadOneImage(imageSrc string, loadOpts *libimage.LoadOptions) error {
	loadedImages, err := engine.ImageRuntime().Load(context.Background(), imageSrc, loadOpts)
	if err != nil {
		return err
	}

	logrus.Infof("Loaded image: " + strings.Join(loadedImages, "\nLoaded image: "))
	return nil
}
