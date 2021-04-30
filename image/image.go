package image

// NewImageService return the image service
func NewImageService() Service {
	return DefaultImageService{
		BaseImageManager{},
	}
}

// NewImageMetadataService return the MetadataService
func NewImageMetadataService() MetadataService {
	return DefaultImageMetadataService{
		BaseImageManager{},
	}
}
