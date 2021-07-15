package charts

import (
	"testing"
)

func TestPackageHelmChart(t *testing.T) {
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
}
