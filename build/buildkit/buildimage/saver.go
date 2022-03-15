package buildimage

import (
	"fmt"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type imageSaver struct {
	buildType  string
	imageStore store.ImageStore
}

func (i imageSaver) Save(image *v1.Image) error {
	err := i.setImageAttribute(image)
	if err != nil {
		return err
	}
	err = i.save(image)
	if err != nil {
		return fmt.Errorf("failed to save image, err: %v", err)
	}

	logger.Info("save image %s success", image.Name)
	return nil
}

func (i imageSaver) setImageAttribute(image *v1.Image) error {
	mi, err := GetLayerMountInfo(image.Spec.Layers, i.buildType)
	if err != nil {
		return err
	}
	defer mi.CleanUp()

	rootfsPath := mi.GetMountTarget()
	is := []ImageSetter{NewAnnotationSetter(rootfsPath), NewPlatformSetter(rootfsPath)}
	for _, s := range is {
		if err = s.Set(image); err != nil {
			return err
		}
	}
	return nil
}

func (i imageSaver) save(image *v1.Image) error {
	imageID, err := generateImageID(*image)
	if err != nil {
		return err
	}
	image.Spec.ID = imageID
	return i.imageStore.Save(*image, image.Name)
}

func NewImageSaver(buildType string) (ImageSaver, error) {
	imageStore, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}
	return imageSaver{
		buildType:  buildType,
		imageStore: imageStore,
	}, nil
}
