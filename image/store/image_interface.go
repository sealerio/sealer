package store

import (
	"github.com/alibaba/sealer/image/types"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type ImageStore interface {
	GetByName(name string) (*v1.Image, error)

	GetByID(id string) (*v1.Image, error)

	DeleteByName(name string) error

	DeleteByID(id string, force bool) error

	Save(image v1.Image, name string) error

	SetImageMetadataItem(name, id string) error

	GetImageMetadataItem(name string) (types.ImageMetadata, error)

	GetImageMetadataMap() (ImageMetadataMap, error)
}
