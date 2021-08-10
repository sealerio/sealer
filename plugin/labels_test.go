package plugin

import (
	"testing"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

func TestLabelsNodes_Run(t *testing.T) {
	type fields struct {
		data map[string][]label
	}

	plugin := &v1.Plugin{}
	plugin.Spec.Data = "192.9.200.21 ws=demo21\n192.9.200.22 ssd=true\n192.9.200.23 ssd=true,hdd=false"

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
			"test label to cluster node",
			fields{},
			args{
				context: Context{
					Plugin: plugin,
				},
				phase: PhasePostInstall,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := LabelsNodes{
				data: tt.fields.data,
			}
			if err := l.Run(tt.args.context, tt.args.phase); (err != nil) != tt.wantErr {
				t.Errorf("LabelsNodes.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
