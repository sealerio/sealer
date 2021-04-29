package image

func NewImageService() Service {
	return DefaultImageService{
		BaseImageManager{},
	}
}

func NewImageMetadataService() MetadataService {
	return DefaultImageMetadataService{
		BaseImageManager{},
	}
}
