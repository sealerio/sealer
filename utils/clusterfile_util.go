package utils

import (
	"fmt"
	"strings"

	"github.com/alibaba/sealer/cert"
	"github.com/alibaba/sealer/logger"

	"io/ioutil"
	"os"
)

func GetDefaultClusterFilePath(clusterFile string) (string, error) {
	if clusterFile == "" {
		return GetDefaultClusterFilePathByClusterName("")
	}
	return clusterFile, nil
}

func GetDefaultClusterFilePathByClusterName(clusterName string) (string, error) {
	sealerPath := fmt.Sprintf("%s/.sealer", cert.GetUserHomeDir())
	if clusterName == "" {
		file, err := GetClusterName(sealerPath)
		if err != nil {
			return "", err
		}
		clusterName = file
	}

	clusterFilePath := fmt.Sprintf("%s/%s/Clusterfile", sealerPath, clusterName)
	if _, err := os.Lstat(clusterFilePath); err != nil {
		return "", err
	}
	return clusterFilePath, nil
}

func GetClusterName(sealerPath string) (string, error) {
	files, err := ioutil.ReadDir(sealerPath)
	if err != nil {
		logger.Error(err)
		return "", err
	}
	var clusters []string
	for _, f := range files {
		if f.IsDir() {
			clusters = append(clusters, f.Name())
		}
	}
	var clusterName string
	if len(clusters) == 1 {
		clusterName = clusters[0]
	} else if len(clusters) > 1 {
		return "", fmt.Errorf("Select a cluster through the -c parameter: " + strings.Join(clusters, ","))
	} else {
		return "", fmt.Errorf("existing cluster not found")
	}
	return clusterName, nil
}
