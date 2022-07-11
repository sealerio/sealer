// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package charts

import (
	"encoding/json"
	"fmt"

	"github.com/sealerio/sealer/build/layerutils"

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

	options := chartutil.ReleaseOptions{
		Name: "dryrun",
	}
	valuesToRender, err := chartutil.ToRenderValues(ch, nil, options, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to render values: %v", err)
	}

	content, err := engine.Render(ch, valuesToRender)
	if err != nil {
		b, _ := json.Marshal(valuesToRender)
		logrus.Debugf("values is %s", b)
		return nil, fmt.Errorf("failed to render helm chart: %s", err)
	}

	return content, nil
}

func GetImageList(chartPath string) ([]string, error) {
	var list []string
	content, err := RenderHelmChart(chartPath)
	if err != nil {
		return list, fmt.Errorf("failed to render helm chart: %s", err)
	}

	for _, v := range content {
		images := layerutils.DecodeImages(v)
		if len(images) != 0 {
			list = append(list, images...)
		}
	}

	return list, nil
}
