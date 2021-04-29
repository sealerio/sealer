package utils

import (
	"gitlab.alibaba-inc.com/seadent/pkg/common"
	"os"
	"path/filepath"
)

func ExecutableFilePath() string {
	ex, _ := os.Executable()
	exPath := filepath.Dir(ex)
	return filepath.Join(exPath, common.ExecBinaryFileName)
}
