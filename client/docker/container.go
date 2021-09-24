package docker

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

func (d Docker) RmContainerByName(containerName string) error {
	containers, err := d.GetContainerListByName(containerName)
	if err != nil {
		return err
	}
	for _, c := range containers {
		err = d.cli.ContainerRemove(d.ctx, c.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (d Docker) GetContainerListByName(containerName string) ([]types.Container, error) {
	opts := types.ContainerListOptions{All: true}
	opts.Filters = filters.NewArgs()
	opts.Filters.Add("name", containerName)
	containers, err := d.cli.ContainerList(d.ctx, opts)

	if err != nil {
		return nil, err
	}

	return containers, err
}
