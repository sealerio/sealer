package apply

import (
	"path/filepath"

	"github.com/onsi/gomega"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/test/testhelper"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

func getFixtures() string {
	pwd := testhelper.GetPwd()
	return filepath.Join(pwd, "suites", "apply", "fixtures")
}

func GetTestClusterFilePath() string {
	fixtures := getFixtures()
	return filepath.Join(fixtures, "cluster_file_for_test.yaml")
}

func DeleteCluster(clusterFile string) {
	cmd := "sudo env PATH=$PATH sealer delete -f " + clusterFile
	testhelper.RunCmdAndCheckResult(cmd, 0)
}

func LoadClusterFileData(clusterFile string) *v1.Cluster {
	cluster := &v1.Cluster{}
	if err := utils.UnmarshalYamlFile(clusterFile, cluster); err != nil {
		logger.Error("failed to unmarshal yamlFile to get clusterFile data")
		return nil
	}
	return cluster
}

func WriteClusterFileToDisk(cluster *v1.Cluster, clusterFilePath string) error {
	if err := utils.MarshalYamlToFile(clusterFilePath, cluster); err != nil {
		logger.Error("failed to write cluster file to disk")
		return err
	}
	return nil
}

func GetClusterNodes() int {
	client, err := testhelper.NewClientSet()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	nodes, err := testhelper.ListNodes(client)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return len(nodes.Items)
}

func CheckClusterPods() int {
	client, err := testhelper.NewClientSet()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	pods, err := testhelper.ListNodes(client)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return len(pods.Items)
}
