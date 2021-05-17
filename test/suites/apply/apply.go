package apply

import (
	"fmt"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"path/filepath"
)

func getFixtures() string {
	pwd := testhelper.GetPwd()
	return filepath.Join(pwd, "suites", "apply", "fixtures")
}

func GetClusterFilePathOfRootfs() string {
	fixtures := getFixtures()
	return filepath.Join(fixtures, "cluster_file_rootfs.yaml")
}

func GetClusterFileData(clusterFile string) *v1.Cluster {
	cluster := &v1.Cluster{}
	if err := utils.UnmarshalYamlFile(clusterFile, cluster); err != nil {
		logger.Error("failed to unmarshal yamlFile to get clusterFile data")
		return nil
	}
	return cluster
}

func DeleteCluster(clusterName string) {
	cmd := "sealer delete -f " + fmt.Sprintf(settings.DefaultClusterFileNeedToBeCleaned, clusterName)
	testhelper.RunCmdAndCheckResult(cmd, 0)
}
