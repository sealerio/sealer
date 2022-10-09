// Copyright © 2022 Alibaba Group Holding Ltd.
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

package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHelm(t *testing.T) {
	t.Run("charts under targets root", func(t *testing.T) {
		dir, err := os.MkdirTemp("/tmp/", "sealer-test")
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			_ = os.RemoveAll(dir)
		}()

		targets := []string{
			filepath.Join(dir, "values.yaml"),
			filepath.Join(dir, "Chart.yaml"),
			filepath.Join(dir, "templates"),
		}
		for _, tar := range targets {
			if _, err := os.Create(tar); err != nil {
				t.Error(err)
				return
			}
		}

		isH, err := isHelm(targets...)
		if err != nil {
			t.Error(err)
			return
		}
		assert.Equal(t, true, isH)
	})

	t.Run("charts under dir of target", func(t *testing.T) {
		githubAppChartsPath := "./test/brigade-github-app"
		isH, err := isHelm(githubAppChartsPath)
		if err != nil {
			t.Error(err)
			return
		}
		assert.Equal(t, true, isH)
	})

	t.Run("no charts under the targets", func(t *testing.T) {
		nginxPath := "./test/kube-nginx-deployment"
		isH, err := isHelm(nginxPath)
		if err != nil {
			t.Error(err)
			return
		}
		assert.Equal(t, false, isH)
	})

	// There is a problem
	// if the charts does not exist in the first layer of targets
	// could it be the charts?
	t.Run("kube and charts exist same time", func(t *testing.T) {
		targets := []string{
			"./test/kube-nginx-deployment",
			"./test/brigade-github-app",
		}

		isH, err := isHelm(targets...)
		if err != nil {
			t.Error(err)
			return
		}
		assert.Equal(t, true, isH)
	})

	t.Run("charts under subdir of target dir", func(t *testing.T) {
		targets := []string{
			"./test/",
		}

		isH, err := isHelm(targets...)
		if err != nil {
			t.Error(err)
			return
		}
		assert.Equal(t, false, isH)
	})
}
