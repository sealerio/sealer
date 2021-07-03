package boot

import (
	"fmt"
	"os"

	"github.com/alibaba/sealer/common"
)

var rootDirs = []string{
	common.DefaultImageRootDir,
	common.DefaultImageMetaRootDir,
	common.DefaultImageDBRootDir,
	common.DefaultLayerDir,
	common.DefaultLayerDBRoot}

func initRootDirectory() error {
	for _, dir := range rootDirs {
		err := os.MkdirAll(dir, common.FileMode0755)
		if err != nil {
			return fmt.Errorf("failed to mkdir %s, err: %s", dir, err)
		}
	}
	return nil
}

func OnBoot() error {
	return initRootDirectory()
}
