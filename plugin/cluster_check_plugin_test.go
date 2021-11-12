package plugin

import (
	"testing"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

func TestClusterCheck_Run(t *testing.T) {
	type fields struct{}

	plugin := &v1.Plugin{}

	type args struct {
		context Context
		phase   Phase
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"test check cluster status",
			fields{},
			args{
				context: Context{
					Plugin: plugin,
				},
				phase: PhasePreGuest,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ClusterChecker{}
			if err := c.Run(tt.args.context, tt.args.phase); (err != nil) != tt.wantErr {
				t.Errorf("clusterCheck.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
