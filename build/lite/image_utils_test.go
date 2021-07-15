package lite

import (
	"reflect"
	"testing"
)

func Test_decodeImages(t *testing.T) {
	body := `
          image: cn-app-integration:v1.0.0
          image: registry.cn-shanghai.aliyuncs.com/cnip/cn-app-integration:v1.0.0
          imagePullPolicy: Always
          image: cn-app-integration:v1.0.0
		  # image: cn-app-integration:v1.0.0
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
			if got := DecodeImages(tt.args.body); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeImages() = %v, want %v", got, tt.want)
			}
		})
	}
}
