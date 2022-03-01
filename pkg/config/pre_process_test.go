package config

import (
	"testing"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

func TestNewProcessorsAndRun(t *testing.T) {
	config := &v1.Config{
		Spec: v1.ConfigSpec{
			Process: "value|toJson|toBase64",
			Data: `
config:
  usrname: root
  passwd: xxx
`,
		},
	}

	type args struct {
		config *v1.Config
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "test value|toJson|toBase64",
			args:    args{config},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NewProcessorsAndRun(tt.args.config); (err != nil) != tt.wantErr || tt.args.config.Spec.Data != "config: eyJwYXNzd2QiOiJ4eHgiLCJ1c3JuYW1lIjoicm9vdCJ9\n" {
				t.Errorf("NewProcessorsAndRun() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
