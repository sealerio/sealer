package plugin

import (
	"testing"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

func TestEtcdBackupPlugin_Run(t *testing.T) {
	type etcdBackup struct {
		name    string
		backDir string
	}
	type args struct {
		context Context
		phase   Phase
	}

	cluster := &v1.Cluster{}
	cluster.Spec.SSH.User = "root"
	cluster.Spec.SSH.Passwd = "123456"
	cluster.Spec.Masters.IPList = []string{"172.17.189.55"}

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
					Cluster: cluster,
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
