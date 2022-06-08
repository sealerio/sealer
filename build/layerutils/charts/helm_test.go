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

/* func TestPackageHelmChart(t *testing.T) {
	type args struct {
		chartPath string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"test package helm chart",
			args{"./test/alpine"},
			"alpine-0.1.0.tgz",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PackageHelmChart(tt.args.chartPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("PackageHelmChart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PackageHelmChart() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderHelmChart(t *testing.T) {
	type args struct {
		chartPath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"test render helm chart",
			args{"./test/alpine"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RenderHelmChart(tt.args.chartPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderHelmChart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}*/
