package charts

import (
	"reflect"
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
			"test package helm chart",
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

func Test_decodeImages(t *testing.T) {
	body := `
          image: cn-app-integration:v1.0.0
          image: registry.cn-shanghai.aliyuncs.com/cnip/cn-app-integration:v1.0.0
          imagePullPolicy: Always
          image: cn-app-integration:v1.0.0
          name: cn-app-demo`
	type args struct {
		body string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			"test get iamges form yaml",
			args{body},
			[]string{"cn-app-integration:v1.0.0", "registry.cn-shanghai.aliyuncs.com/cnip/cn-app-integration:v1.0.0", "cn-app-integration:v1.0.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeImages(tt.args.body); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeImages() = %v, want %v", got, tt.want)
			}
		})
	}
}
