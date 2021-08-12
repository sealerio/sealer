package plugin

import (
	"testing"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

func TestDumperPlugin_Dump(t *testing.T) {
	type fields struct {
		configs     []v1.Plugin
		clusterName string
	}
	plugins := []v1.Plugin{
		{
			Spec: v1.PluginSpec{
				On:     "role=master",
				Data:   "kubectl taint nodes node-role.kubernetes.io/master=:NoSchedule",
				Action: "PostInstall",
			},
		},
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
			name: "Test",
			fields: fields{
				configs:     plugins,
				clusterName: "my-cluster",
			},
			args: args{
				clusterfile: "test_clusterfile.yaml",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PluginsProcesser{
				plugins:     tt.fields.configs,
				clusterName: tt.fields.clusterName,
			}
			if err := c.Dump(tt.args.clusterfile); (err != nil) != tt.wantErr {
				t.Errorf("Dump() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDumperPlugin_Run(t *testing.T) {
	type fields struct {
		configs     []v1.Plugin
		clusterName string
	}
	type args struct {
		cluster *v1.Cluster
		phase   Phase
	}
	plugins := []v1.Plugin{
		{
			Spec: v1.PluginSpec{
				On:     "role=master",
				Data:   "kubectl taint nodes node-role.kubernetes.io/master=:NoSchedule",
				Action: "PostInstall",
			},
		},
	}
	//TODO cluster is where?
	cluster := &v1.Cluster{}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test",
			fields: fields{
				configs:     plugins,
				clusterName: "my-cluster",
			},
			args: args{
				cluster: cluster,
				phase:   "PostInstall",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PluginsProcesser{
				plugins:     tt.fields.configs,
				clusterName: tt.fields.clusterName,
			}
			if err := c.Run(tt.args.cluster, tt.args.phase); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
