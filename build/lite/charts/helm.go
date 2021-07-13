package charts

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

func Load(chartPath string) (*chart.Chart, error) {
	return loader.LoadDir(chartPath)
}

func PackageHelmChart(chartPath string) (string, error) {
	ch, err := Load(chartPath)
	if err != nil {
		return "", err
	}

	name, err := chartutil.Save(ch, ".")
	if err != nil {
		return "", err
	}

	return name, nil
}

func RenderHelmChart(chartPath string) (map[string]string, error) {
	ch, err := Load(chartPath)
	if err != nil {
		return nil, err
	}

	/*
		values := map[string]interface{}{
			"Release": map[string]interface{}{
				"Name": "dryrun",
			},
			"Values": ch.Values,
		}
	*/
	options := chartutil.ReleaseOptions{
		Name: "dryrun",
	}
	valuesToRender, err := chartutil.ToRenderValues(ch, nil, options, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to render values %v", err)
	}

	content, err := engine.Render(ch, valuesToRender)
	if err != nil {
		b, _ := json.Marshal(valuesToRender)
		logrus.Debugf("values is %s", b)
		return nil, fmt.Errorf("render helm chart error %s", err.Error())
	}

	//for k, v := range content {
	//	fmt.Println(k, v)
	//}

	return content, nil
}

func GetImageList(chartPath string) ([]string, error) {
	var list []string
	content, err := RenderHelmChart(chartPath)
	if err != nil {
		return list, fmt.Errorf("render helm chart failed %s", err)
	}

	for _, v := range content {
		images := decodeImages(v)
		if len(images) != 0 {
			list = append(list, images...)
		}
	}

	return list, nil
}

// decode image from yaml content
func decodeImages(body string) []string {
	var list []string

	reader := strings.NewReader(body)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		l := decodeLine(scanner.Text())
		if l != "" {
			list = append(list, l)
		}
	}
	if err := scanner.Err(); err != nil {
		logrus.Errorf(err.Error())
		return list
	}

	return list
}

func decodeLine(line string) string {
	l := strings.Replace(line, `"`, "", -1)
	ss := strings.SplitN(l, ":", 2)
	if len(ss) != 2 {
		return ""
	}
	if !strings.HasSuffix(ss[0], "image") || strings.Contains(ss[0], "#") {
		return ""
	}

	return strings.Replace(ss[1], " ", "", -1)
}
