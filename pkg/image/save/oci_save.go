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

package save

import (
	"context"
	"fmt"
	"io"
	"strings"

	dockerstreams "github.com/docker/cli/cli/streams"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/image/save/skopeo"
	v1 "github.com/sealerio/sealer/types/api/v1"
	osi "github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/os/fs"
)

// isImageExist check if an image exist in specified storage
func (is *OCIImageSaver) isImageExist(prefix string, named Named, dir string) bool {
	imageName := getImageNameWithStorageType(prefix, named, dir)
	return skopeo.IsImageExist(imageName)
}

func (is *OCIImageSaver) SaveImages(images []string, dir string, platform v1.Platform) error {
	//init a pipe for display pull message
	reader, writer := io.Pipe()
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
	}()
	is.progressOut = streamformatter.NewJSONProgressOutput(writer, false)

	go func() {
		err := dockerjsonmessage.DisplayJSONMessagesToStream(reader, dockerstreams.NewOut(common.StdOut), nil)
		if err != nil && err != io.ErrClosedPipe {
			logrus.Warnf("error occurs in display progressing, err: %s", err)
		}
	}()

	//handle image name
	for _, image := range images {
		named, err := ParseNormalizedNamed(image, "")
		if err != nil {
			return fmt.Errorf("failed to parse image name:: %v", err)
		}
		//check if image exist
		if is.isImageExist(skopeo.OciPath, named, dir) {
			continue
		}
		//check if docker-daemon or containers-storage has cache
		if is.isImageExist(skopeo.DockerDaemon, named, "") {
			is.typeToImages[skopeo.DockerDaemon] = append(is.typeToImages[skopeo.DockerDaemon], named)
		} else if is.isImageExist(skopeo.ContainersStorage, named, "") {
			is.typeToImages[skopeo.ContainersStorage] = append(is.typeToImages[skopeo.ContainersStorage], named)
		} else {
			is.typeToImages[skopeo.RemoteRegistry] = append(is.typeToImages[skopeo.RemoteRegistry], named)
		}
		progress.Message(is.progressOut, "", fmt.Sprintf("Pulling image: %s", named.FullName()))
	}

	//perform image save ability
	eg, _ := errgroup.WithContext(context.Background())
	numCh := make(chan struct{}, maxPullGoroutineNum)
	for imageType, nameds := range is.typeToImages {
		tmpnameds := nameds
		tmpType := imageType
		numCh <- struct{}{}
		eg.Go(func() error {
			defer func() {
				<-numCh
			}()
			err := is.save(tmpType, tmpnameds, "", dir)
			if err != nil {
				return fmt.Errorf("ImageType:%s failed to save domain %s image: %v", imageType, tmpnameds[0].domain, err)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	if len(images) != 0 {
		progress.Message(is.progressOut, "", "Status: images save success")
	}
	return nil
}

func (is *OCIImageSaver) SaveImagesWithAuth(imageList ImageListWithAuth, dir string, platform v1.Platform) error {
	//init a pipe for display pull message
	reader, writer := io.Pipe()
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
	}()
	is.progressOut = streamformatter.NewJSONProgressOutput(writer, false)
	is.ctx = context.Background()
	go func() {
		err := dockerjsonmessage.DisplayJSONMessagesToStream(reader, dockerstreams.NewOut(common.StdOut), nil)
		if err != nil && err != io.ErrClosedPipe {
			logrus.Warnf("error occurs in display progressing, err: %s", err)
		}
	}()

	//perform image save ability
	eg, _ := errgroup.WithContext(context.Background())
	numCh := make(chan struct{}, maxPullGoroutineNum)

	//handle imageList
	for _, section := range imageList {
		for _, nameds := range section.Images {
			tmpnameds := nameds
			creds := section.Username + ":" + section.Password
			progress.Message(is.progressOut, "", fmt.Sprintf("Pulling image: %s", tmpnameds[0].FullName()))
			numCh <- struct{}{}
			eg.Go(func() error {
				defer func() {
					<-numCh
				}()
				if err := is.save(skopeo.RemoteRegistry, tmpnameds, creds, dir); err != nil {
					return err
				}
				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			return err
		}
	}

	if len(imageList) != 0 {
		progress.Message(is.progressOut, "", "Status: images save success")
	}
	return nil
}

func (is *OCIImageSaver) save(imageType string, namds []Named, creds string, dir string) error {
	for _, namd := range namds {
		srcImageName := getImageNameWithStorageType(imageType, namd, "")
		destImageName := getImageNameWithStorageType(skopeo.OciPath, namd, dir)
		if err := getDestDir(destImageName); err != nil {
			return err
		}
		err := skopeo.Copy(srcImageName, destImageName, creds)
		if err != nil {
			return err
		}
	}
	return nil
}

// getImageNameWithStorageType get args of `skopeo copy`
func getImageNameWithStorageType(prefix string, named Named, dir string) string {
	if dir == "" { //
		return prefix + named.FullName()
	}
	registry := dir
	imageName := named.repo + ":" + named.tag
	return prefix + registry + "/" + imageName
}
func getDestDir(destImage string) error {
	path := strings.Split(destImage, ":")
	dir := strings.Split(path[1], "/")
	registryDir := strings.Join(dir[:len(dir)-1], "/")
	if _, err := fs.FS.Stat(registryDir); err == nil {
		return nil
	}
	if osi.IsFileExist(registryDir) {
		return nil
	}
	if err := fs.FS.MkdirAll(registryDir); err != nil {
		return err
	}
	return nil
}
