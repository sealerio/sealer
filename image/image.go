package image

// NewImageService return the image service
func NewImageService() Service {
	return DefaultImageService{}
}

// NewImageMetadataService return the MetadataService
func NewImageMetadataService() MetadataService {
	return DefaultImageMetadataService{}
}

// NewImageFileService return the file Service
func NewImageFileService() FileService {
	return DefaultImageFileService{}
}
