package build

import (
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/opencontainers/go-digest"
	"sigs.k8s.io/yaml"
)

func upImageId(imageName string, Image *v1.Image) error {
	var (
		imageBytes []byte
		imageStore store.ImageStore
		err        error
	)
	imageBytes, err = yaml.Marshal(Image)
	if err != nil {
		return err
	}
	imageID := digest.FromBytes(imageBytes).Hex()
	Image.Spec.ID = imageID
	imageStore, err = store.NewDefaultImageStore()
	if err != nil {
		return err
	}
	return imageStore.Save(*Image, imageName)
}

func Merge(imageName string, images []string) error {
	var Image = &v1.Image{}
	for k, v := range images {
		img, err := image.DefaultImageService{}.PullIfNotExistAndReturnImage(v)
		if err != nil {
			return err
		}
		if k == 0 {
			Image = img
			Image.Name = imageName
			Image.Spec.Layers = img.Spec.Layers
		} else {
			Image.Spec.Layers = append(Image.Spec.Layers, img.Spec.Layers[1:]...)
		}
	}
	return upImageId(imageName, Image)
}
