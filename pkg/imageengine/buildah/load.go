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
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/utils/archive"
	"github.com/sirupsen/logrus"
)

var LoadError = errors.Errorf("failed to load new image")

func (engine *Engine) Load(opts *options.LoadOptions) error {
	imageSrc := opts.Input
	if _, err := os.Stat(imageSrc); err != nil {
		return err
	}

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
	if _, err := os.Stat(metaFile); err != nil {
		//assume it is single image to load
		return engine.loadOneImage(imageSrc, loadOpts)
	}

	// get manifestName
	metaBytes, err := ioutil.ReadFile(filepath.Clean(metaFile))
	if err != nil {
		return err
	}

	manifestName := string(metaBytes)
	// delete it if manifestName is already used
	_, err = engine.ImageRuntime().LookupManifestList(manifestName)
	if err == nil {
		logrus.Warnf("%s is already in use, will delete it", manifestName)
		delErr := engine.DeleteManifests([]string{manifestName}, &options.ManifestDeleteOpts{})
		if delErr != nil {
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

	err = engine.AddToManifest(manifestName, instancesIDs, &options.ManifestAddOpts{})
	if err != nil {
		return fmt.Errorf("failed to add new image to %s :%v ", manifestName, err)
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
