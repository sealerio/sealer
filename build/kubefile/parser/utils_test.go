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

	"github.com/sealerio/sealer/pkg/define/application"

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

func TestGetApplicationType(t *testing.T) {
	type args struct {
		preparedFunc func() (string, error)
		wanted       string
	}
	var tests = []struct {
		name string
		args args
	}{
		{
			name: "it is helm application",
			args: args{
				preparedFunc: func() (string, error) {
					dir, err := os.MkdirTemp("/tmp/", "sealer-test")
					if err != nil {
						t.Error(err)
					}
					targets := []string{
						filepath.Join(dir, "values.yaml"),
						filepath.Join(dir, "Chart.yaml"),
						filepath.Join(dir, "templates"),
					}
					for _, tar := range targets {
						if _, err := os.Create(tar); err != nil {
							t.Error(err)
						}
					}

					return dir, nil
				},
				wanted: application.HelmApp,
			},
		},
		{
			name: "it is kube yaml application",
			args: args{
				preparedFunc: func() (string, error) {
					dir, err := os.MkdirTemp("/tmp/", "sealer-test")
					if err != nil {
						t.Error(err)
					}
					targets := []string{
						filepath.Join(dir, "a.yaml"),
						filepath.Join(dir, "b.yaml"),
						filepath.Join(dir, "c.yaml"),
					}
					for _, tar := range targets {
						if _, err := os.Create(tar); err != nil {
							t.Error(err)
						}
					}

					return dir, nil
				},
				wanted: application.KubeApp,
			},
		},
		{
			name: "it is shell application",
			args: args{
				preparedFunc: func() (string, error) {
					dir, err := os.MkdirTemp("/tmp/", "sealer-test")
					if err != nil {
						t.Error(err)
					}
					targets := []string{
						filepath.Join(dir, "a.sh"),
						filepath.Join(dir, "v.sh"),
						filepath.Join(dir, "c.sh"),
					}
					for _, tar := range targets {
						if _, err := os.Create(tar); err != nil {
							t.Error(err)
						}
					}

					return dir, nil
				},
				wanted: application.ShellApp,
			},
		},
		{
			name: "it is mixed application",
			args: args{
				preparedFunc: func() (string, error) {
					dir, err := os.MkdirTemp("/tmp/", "sealer-test")
					if err != nil {
						t.Error(err)
					}
					targets := []string{
						filepath.Join(dir, "a.sh"),
						filepath.Join(dir, "t.yaml"),
						filepath.Join(dir, "c.sh"),
					}
					for _, tar := range targets {
						if _, err := os.Create(tar); err != nil {
							t.Error(err)
						}
					}

					return dir, nil
				},
				wanted: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := tt.args.preparedFunc()
			if err != nil {
				assert.Errorf(t, err, "failed to create test files for %s: %v", tt.name, err)
			}
			defer func() {
				_ = os.RemoveAll(dir)
			}()

			appType, err := getApplicationType([]string{dir})
			if err != nil {
				t.Error(err)
				return
			}

			assert.Equal(t, tt.args.wanted, appType)
		})
	}
}
