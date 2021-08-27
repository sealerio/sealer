package debug

import (
	"fmt"

	"github.com/spf13/cobra"
)

type DebugImagesManagement interface {
	ShowDefaultImages() error
	GetDefaultImage() (string, error)
}
const SealerRegistryUrl = "registry.cn-qingdao.aliyuncs.com/sealer-apps/"

// DebugImagesManager holds the default images information.
type DebugImagesManager struct {
	RegistryURL			string

	DefaultImagesMap	map[string]string	// "RichToolsOnUbuntu": "debug:ubuntu"
	DefaultImageKey		string				// "RichToolsOnUbuntu"

	DefaultImage		string				// RegistryURL + DefaultImagesMap[DefaultImageName]
}

func NewDebugImagesManager() *DebugImagesManager {
	return &DebugImagesManager{
		DefaultImagesMap: map[string]string{
			"RichToolsOnUbuntu": "debug:ubuntu",
		},

		DefaultImageKey:  "RichToolsOnUbuntu",
	}
}

func NewDebugShowImagesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show-images",
		Short:   "List default images",
		Args: 	 cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := NewDebugImagesManager()
			manager.RegistryURL = SealerRegistryUrl

			if err := manager.ShowDefaultImages(); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}

// ShowDefaultImages shows default images provided by debug.
func (manager *DebugImagesManager) ShowDefaultImages() error {
	if len(manager.RegistryURL) == 0 {
		manager.RegistryURL = SealerRegistryUrl
	}
	fmt.Println("There are several default images you can use：")
	for key, value := range manager.DefaultImagesMap {
		fmt.Println(key + ":  " + manager.RegistryURL + value)
	}

	return nil
}

// GetDefaultImage return the default image provide by debug.
func (manager *DebugImagesManager) GetDefaultImage() (string, error) {
	if len(manager.RegistryURL) == 0 {
		manager.RegistryURL = SealerRegistryUrl
	}
	return manager.RegistryURL + manager.DefaultImagesMap[manager.DefaultImageKey], nil
}