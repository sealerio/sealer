package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"gitlab.alibaba-inc.com/seadent/pkg/image"
	"gitlab.alibaba-inc.com/seadent/pkg/logger"
)

var imagePullFlag *ImageFlag

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "pull cloud image to local",
	Long:  `seautil pull my-kubernetes:1.18.3`,
	Run: func(cmd *cobra.Command, args []string) {
		err := image.NewImageService().Pull(imagePullFlag.ImageName)
		if err != nil {
			logger.Error(err)
			os.Exit(-1)
		}
	},
}

func init() {
	imagePullFlag = &ImageFlag{}
	rootCmd.AddCommand(pullCmd)
	pullCmd.Flags().StringVarP(&imagePullFlag.Username, "username", "u", ".", "user name for login registry")
	pullCmd.Flags().StringVarP(&imagePullFlag.Passwd, "passwd", "p", "", "password for login registry")
	pullCmd.Flags().StringVarP(&imagePullFlag.ImageName, "imageName", "t", "", "name of cloud image")
}
