package plugin

import (
	v1 "github.com/alibaba/sealer/types/api/v1"
	"testing"
)

func TestEtcdBackupPlugin_Run(t *testing.T) {
	plugin := &v1.Plugin{}

	type etcdBackup struct {
		name    string
		backDir string
	}
	type args struct {
		context Context
		phase   Phase
	}
	tests := []struct {
		name    string
		fields  etcdBackup
		args    args
		wantErr bool
	}{
		{
			"test label to cluster node",
			etcdBackup{
				name:    "202108112058.bak",
				backDir: "/tmp",
			},
			args{
				context: Context{
					Plugin: plugin,
					Cluster: &v1.Cluster{
						Spec: v1.ClusterSpec{
							Masters: v1.Hosts{
								IPList: []string{"172.17.189.55"},
							},
							SSH: v1.SSH{
								User:   "root",
								Passwd: "123456",
							},
						},
					},
				},
				phase: PhasePostInstall,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := EtcdBackupPlugin{
				name:    tt.fields.name,
				backDir: tt.fields.backDir,
			}
			if err := e.Run(tt.args.context, tt.args.phase); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
