package apply

import (
	"fmt"
	"path/filepath"

	"github.com/alibaba/sealer/test/testhelper"
)

func getFixtures() string {
	pwd := testhelper.GetPwd()
	return filepath.Join(pwd, "suites", "apply", "fixtures")
}

func GetClusterFilePathOfRootfs() string {
	fixtures := getFixtures()
	return filepath.Join(fixtures, "cluster_file_rootfs.yaml")
}

func DoApplyOrDelete(action, clusterFilePath string) {
	cmd := ""
	if action == "apply" {
		cmd = fmt.Sprintf("sealer apply -f %s", clusterFilePath)
	}

	if action == "delete" {
		cmd = fmt.Sprintf("sealer delete -f %s", clusterFilePath)
	}
	testhelper.RunCmdAndCheckResult(cmd, 0)
}
