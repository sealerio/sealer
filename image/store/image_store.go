package store

import (
	"fmt"

	"github.com/alibaba/sealer/image/types"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

type imageStore struct {
	backend Backend
}

func (is *imageStore) GetByName(name string) (*v1.Image, error) {
	return is.backend.getImageByName(name)
}

func (is *imageStore) GetByID(id string) (*v1.Image, error) {
	return is.backend.getImageByID(id)
}

func (is *imageStore) DeleteByName(name string) error {
	return is.backend.deleteImage(name)
}

func (is *imageStore) DeleteByID(id string, force bool) error {
	return is.backend.deleteImageByID(id, force)
}

func (is *imageStore) Save(image v1.Image, name string) error {
	return is.backend.saveImage(image, name)
}

func (is *imageStore) SetImageMetadataItem(name, id string) error {
	return is.backend.setImageMetadata(types.ImageMetadata{Name: name, ID: id})
}

func (is *imageStore) GetImageMetadataItem(name string) (types.ImageMetadata, error) {
	return is.backend.getImageMetadataItem(name)
}

func (is *imageStore) GetImageMetadataMap() (ImageMetadataMap, error) {
	return is.backend.getImageMetadataMap()
}

func NewDefaultImageStore() (ImageStore, error) {
	backend, err := NewFSStoreBackend()
	if err != nil {
		return nil, fmt.Errorf("failed to init fs store backend, err: %v", err)
	}

	return &imageStore{
		backend: backend,
	}, nil
}
