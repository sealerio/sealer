package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"gitlab.alibaba-inc.com/seadent/pkg/image"
	"gitlab.alibaba-inc.com/seadent/pkg/logger"
)

type ImageFlag struct {
	RegistryURL string
	Username    string
	Passwd      string
	ImageName   string
}

var imageFlag *ImageFlag

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "push cloud image to registry",
	Long:  `seautil push my-kubernetes:1.18.3`,
	Run: func(cmd *cobra.Command, args []string) {
		err := image.NewImageService().Push(imageFlag.ImageName)
		if err != nil {
			logger.Error(err)
			os.Exit(-1)
		}
	},
}

func init() {
	imageFlag = &ImageFlag{}
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().StringVarP(&imageFlag.Username, "username", "u", ".", "user name for login registry")
	pushCmd.Flags().StringVarP(&imageFlag.Passwd, "passwd", "p", "", "password for login registry")
	pushCmd.Flags().StringVarP(&imageFlag.ImageName, "imageName", "t", "", "name of cloud image")
}
