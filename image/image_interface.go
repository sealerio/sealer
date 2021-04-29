package image

import (
	"gitlab.alibaba-inc.com/seadent/pkg/image/utils"
	v1 "gitlab.alibaba-inc.com/seadent/pkg/types/api/v1"
)

type MetadataService interface {
	Tag(imageName, tarImageName string) error
	List() ([]utils.ImageMetadata, error)
	GetImage(imageName string) (*v1.Image, error)
	GetRemoteImage(imageName string) (v1.Image, error)
}

type FileService interface {
	Load(imageSrc string) error
	Save(imageName string, imageTar string) error
	Merge(image *v1.Image) error
}

type Service interface {
	Pull(imageName string) error
	PullIfNotExist(imageName string) error
	Push(imageName string) error
	Login(RegistryURL, RegistryUsername, RegistryPasswd string) error
}
