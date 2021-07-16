package types

import "fmt"

type ImageNameOrIDNotFoundError struct {
	Name string
}

func (e *ImageNameOrIDNotFoundError) Error() string {
	return fmt.Sprintf("failed to find imageName or imageId %s", e.Name)
}
