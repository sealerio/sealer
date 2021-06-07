package utils

import "fmt"

type ImageNameOrIDNotFoundError struct {
	name string
}

func (e *ImageNameOrIDNotFoundError) Error() string {
	return fmt.Sprintf("failed to find imageName or imageId %s", e.name)
}
