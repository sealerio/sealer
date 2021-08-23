package debug

import (
	"fmt"

	"github.com/spf13/cobra"
)

var DEFAULT_IMAGES_MAP = map[string]string{
	"RichToolsOnUbuntu": "debug:ubuntu",
	"RichToolsOnAlpine": "debug:apline",
}

type ImagesOptions struct {
	RegistryURL			string

	defaultImagesMap	map[string]string	// "RichToolsOnUbuntu": "debug:ubuntu"
	defaultImageKey		string				// "RichToolsOnUbuntu"
	defaultImageURL		string				// RegistryURL + DefaultImagesMap[DefaultImageName]
}

func NewImagesOptions() *ImagesOptions {
	return &ImagesOptions{
		defaultImagesMap: 	DEFAULT_IMAGES_MAP,
		defaultImageKey: 	"RichToolsOnUbuntu",
	}
}

func NewDebugImages() *cobra.Command {
	imagesOptions := NewImagesOptions()

	cmd := &cobra.Command{
		Use:     "show-images",
		Short:   "List default images",
		Long:    "",
		Example: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := imagesOptions.ShowDefaultImages(); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}

// ShowDefaultImages shows default images provided by debug.
func (imgOpts *ImagesOptions) ShowDefaultImages() error {
	url, err := imgOpts.GetRegistryURL()
	if err != nil {
		return err
	}
	imgOpts.RegistryURL =  url

	fmt.Println("There are several default images you can use：")
	for key, value := range imgOpts.defaultImagesMap {
		fmt.Println(key + ":  " + imgOpts.RegistryURL + value)
	}

	return nil
}

// GetDefaultImage return the default image provide by debug.
func (imgOpts *ImagesOptions) GetDefaultImage() (string, error) {
	url, err := imgOpts.GetRegistryURL()
	if err != nil {
		return "", err
	}

	return url + imgOpts.defaultImagesMap[imgOpts.defaultImageKey], nil
}

// GetRegistryURL returns the registry url.
func (imgOpts *ImagesOptions) GetRegistryURL() (string, error) {
	if len(imgOpts.RegistryURL) != 0 {
		return imgOpts.RegistryURL, nil
	}

	// Diff: between trident and sealer
	//return GetTridentRegistryUrl()
	return GetSealerRegistryUrl()
}

// GetTridentRegistryUrl returns the default registry url.
//func GetTridentRegistryUrl() (string, error) {
//	cluster, err := parse.LoadActualClusterFromAPIServer(3, 1*time.Second);
//	if err != nil {
//		return "", errors.Wrapf(err, "failed to load current cluster for k8s api server")
//	}
//
//	return cluster.Spec.LocalRegistry.URL + ":" + strconv.Itoa(cluster.Spec.LocalRegistry.Port) + "/oecp/", nil
//}

// GetSealerRegistryUrl returns the default registry url.
func GetSealerRegistryUrl() (string, error) {
	return "", nil
}


