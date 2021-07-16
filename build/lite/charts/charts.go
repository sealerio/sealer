package charts

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/alibaba/sealer/build/lite"
	"github.com/alibaba/sealer/common"
)

type Charts struct{}

// List all the containers images in helm charts
func (charts *Charts) ListImages(clusterName string) ([]string, error) {
	var list []string

	chartsRootDir := defaultChartsRootDir(clusterName)
	files, err := ioutil.ReadDir(chartsRootDir)
	if err != nil {
		return list, fmt.Errorf("list images failed %s", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			// skip files
			continue
		}
		chartPath := filepath.Join(chartsRootDir, file.Name())
		images, err := GetImageList(chartPath)
		if err != nil {
			return list, fmt.Errorf("get images failed,chart path:%s, err: %s", chartPath, err)
		}
		if len(images) != 0 {
			list = append(list, images...)
		}
	}

	return list, nil
}

func NewCharts() (lite.Interface, error) {
	return &Charts{}, nil
}

func defaultChartsRootDir(clusterName string) string {
	return filepath.Join(common.DefaultTheClusterRootfsDir(clusterName), "charts")
}
