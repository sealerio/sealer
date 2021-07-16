package container

import (
	"fmt"
	"io"
	"os"

	"github.com/alibaba/sealer/logger"
	"github.com/docker/docker/api/types"
)

func (c *DockerProvider) DeleteImageResource(imageId string) error {
	_, err := c.DockerClient.ImageRemove(c.Ctx, imageId, types.ImageRemoveOptions{
		Force:         true,
		PruneChildren: true,
	})
	return err
}

func (c *DockerProvider) PrepareImageResource() error {
	// if exist, only set id no need to pull
	if imageId := c.GetImageIdByName(c.ImageResource.DefaultName); imageId != "" {
		logger.Info("image %s already exists", c.ImageResource.DefaultName)
		c.ImageResource.Id = imageId
		return nil
	}
	reader, err := c.DockerClient.ImagePull(c.Ctx, c.ImageResource.DefaultName, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		logger.Fatal(err, "unable to read image pull response")
	}
	imageId := c.GetImageIdByName(c.ImageResource.DefaultName)
	if imageId != "" {
		c.ImageResource.Id = imageId
		return nil
	}

	return fmt.Errorf("failed to pull image:%s", c.ImageResource.DefaultName)
}

func (c *DockerProvider) GetImageIdByName(name string) string {
	images, err := c.DockerClient.ImageList(c.Ctx, types.ImageListOptions{})
	if err != nil {
		return ""
	}
	for _, ima := range images {
		named := ima.RepoTags
		for _, imaName := range named {
			if imaName == name {
				return ima.ID
			}
		}
	}
	return ""
}

func (c *DockerProvider) GetImageResourceById(id string) (*types.ImageInspect, error) {
	image, _, err := c.DockerClient.ImageInspectWithRaw(c.Ctx, id)
	return &image, err
}
