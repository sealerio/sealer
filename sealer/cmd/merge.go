package cmd

import (
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var ImageName string

var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge multiple images into one",
	Long:  `sealer merge image1:latest image2:latest image3:latest ......`,
	Example: `
merge images:
	sealer merge kubernetes:v1.19.9 mysql:5.7.0 redis:6.0.0  
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var images []string
		for _, v := range strings.Split(args[0], " "){
			image := strings.TrimSpace(v)
			if image == "" {
				continue
			}
			images = append(images, image)
		}
		if err := image.Merge(ImageName, images);err != nil{
			logger.Error(err)
			os.Exit(1)
		}
		logger.Info("images %s is merged to %s!", strings.Join(images, ","), ImageName)
	},
}

func init() {
	rootCmd.AddCommand(mergeCmd)
	rootCmd.Flags().StringVarP(&ImageName, "imageName", "t", "", "target image name")
}

