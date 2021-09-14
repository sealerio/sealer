package image

import (
	"fmt"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/mount"
	"os"
	"path/filepath"
)

func Merge(imageName string, images []string) error {
	var (
		driver  = mount.NewMountDriver()
		err     error
		Layers  []v1.Layer
		layers  []string
		tempDir string
	)
	for i, v := range images {
		image, err := GetImageByName(v)
		if err != nil {
			return err
		}
		if i == 0 {
			Layers = append(Layers, image.Spec.Layers...)
		} else {
			Layers = append(Layers, image.Spec.Layers[1:]...)
		}
	}
	tempDir, err = utils.MkTmpdir()
	defer utils.CleanDir(tempDir)

	for _, layer := range Layers {
		if layer.ID != "" {
			layers = append(layers, filepath.Join(tempDir, layer.ID.Hex()))
		}
	}
	if err = os.MkdirAll(tempDir, 0744); err != nil {
		return fmt.Errorf("create upperdir failed, %s", err)
	}
	if err = driver.Mount(tempDir, filepath.Join(tempDir, "upper"), layers...); err != nil {
		return fmt.Errorf("mount files failed %v", err)
	}
}
