package store

import (
	"os"
	"testing"

	"github.com/alibaba/sealer/common"

	v1 "github.com/alibaba/sealer/types/api/v1"
	"gotest.tools/skip"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var images = []v1.Image{
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "a",
		},
		Spec: v1.ImageSpec{
			ID: "imagea",
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "b",
		},
		Spec: v1.ImageSpec{
			ID: "imageb",
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "c",
		},
		Spec: v1.ImageSpec{
			ID: "imagec",
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "d",
		},
		Spec: v1.ImageSpec{
			ID: "imaged",
		},
	},
}

var dirs = []string{
	imageDBRoot,
	common.DefaultTmpDir,
}

func init() {
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			panic(err)
		}
	}
}

func TestImageStore_GetImage(t *testing.T) {
	skip.If(t, os.Getuid() != 0, "skipping test that requires root")

	is, err := NewDefaultImageStore()
	if err != nil {
		t.Error(err)
	}

	for _, image := range images {
		err = is.Save(image, image.Name)
		if err != nil {
			t.Errorf("failed to save image %s, err: %s", image.Name, err)
		}
	}

	for _, image := range images {
		_, err = is.GetByID(image.Spec.ID)
		if err != nil {
			t.Errorf("failed to get image by id %s, err: %s", image.Spec.ID, err)
		}

		_, err = is.GetByName(image.Name)
		if err != nil {
			t.Errorf("failed to get image by name %s, err: %s", image.Name, err)
		}

		_, err = is.GetImageMetadataItem(image.Name)
		if err != nil {
			t.Errorf("failed to get image metadata item for %s, err: %s", image.Name, err)
		}
	}
}

func TestImageStore_ImageMetadataItem(t *testing.T) {
	skip.If(t, os.Getuid() != 0, "skipping test that requires root")

	is, err := NewDefaultImageStore()
	if err != nil {
		t.Error(err)
	}

	for _, image := range images {
		err = is.SetImageMetadataItem(image.Name, image.Spec.ID)
		if err != nil {
			t.Errorf("failed to set image metadata for %s, err: %s", image.Name, err)
		}
	}

	for _, image := range images {
		_, err = is.GetImageMetadataItem(image.Name)
		if err != nil {
			t.Errorf("failed to set image metadata for %s, err: %s", image.Name, err)
		}
	}
}
