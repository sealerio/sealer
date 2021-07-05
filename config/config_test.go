package config

import (
	"testing"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

func TestDumper_Dump(t *testing.T) {
	type fields struct {
		configs     []v1.Config
		clusterName string
	}
	type args struct {
		clusterfile string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"test dump clusterfile configs",
			fields{
				configs:     nil,
				clusterName: "my-cluster",
			},
			args{clusterfile: "test_clusterfile.yaml"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Dumper{
				configs:     tt.fields.configs,
				clusterName: tt.fields.clusterName,
			}
			if err := c.Dump(tt.args.clusterfile); (err != nil) != tt.wantErr {
				t.Errorf("Dump() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
