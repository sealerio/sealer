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

package buildstorage

import (
	"github.com/alibaba/sealer/pkg/image"
	"github.com/alibaba/sealer/pkg/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type localFile struct {
	saveName    string
	imageStore  store.ImageStore
	fileService image.FileService
}

func (l localFile) Save(image *v1.Image) error {
	err := l.imageStore.Save(*image, image.Name)
	if err != nil {
		return err
	}

	err = l.fileService.Save(image, l.saveName)
	if err != nil {
		return err
	}
	return nil
}

func NewLocalFile(saveName string) (ImageSaver, error) {
	fs, err := image.NewImageFileService()
	if err != nil {
		return nil, err
	}

	is, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}

	return localFile{
		saveName:    saveName,
		fileService: fs,
		imageStore:  is,
	}, nil
}

type localFileFactory struct{}

func (factory *localFileFactory) Create(parameters map[string]string) (ImageSaver, error) {
	is, err := NewLocalFile(parameters["dest"])
	if err != nil {
		return nil, err
	}
	return is, nil
}

func init() {
	Register(LocalFileFactory, &localFileFactory{})
}
