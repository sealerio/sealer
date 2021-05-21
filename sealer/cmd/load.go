package cmd

import (
	"os"

	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
)

var imageSrc string

// loadCmd represents the load command
var loadCmd = &cobra.Command{
	Use:     "load",
	Short:   "load image",
	Long:    `Load an image from a tar archive`,
	Example: `sealer load -i kubernetes.tar.gz`,
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if err := image.NewImageFileService().Load(imageSrc); err != nil {
			logger.Error("failed to load image from %s, err: %v", imageSrc, err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(loadCmd)
	loadCmd.Flags().StringVarP(&imageSrc, "input", "i", "", "read image from tar archive file")
	if err := loadCmd.MarkFlagRequired("input"); err != nil {
		logger.Error("failed to init flag: %v", err)
		os.Exit(1)
	}
}
