package filesystem

import (
	v1 "gitlab.alibaba-inc.com/seadent/pkg/types/api/v1"
	v2 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestMount(t *testing.T) {
	type args struct {
		cluster *v1.Cluster
	}
	tests := []struct {
		name    string
		arg     args
		wantErr bool
	}{
		{
			name: "test mount",
			arg: args{
				cluster: &v1.Cluster{
					ObjectMeta: v2.ObjectMeta{
						Name: "cluster",
					},
					Spec: v1.ClusterSpec{
						Image: "kuberentes:v1.18.6",
						Masters: v1.Hosts{
							IPList: []string{
								"192.168.56.111",
							},
						},
						Nodes: v1.Hosts{
							IPList: []string{
								"192.168.56.112",
							},
						},
						SSH: v1.SSH{
							User:   "root",
							Passwd: "huaijiahui.com",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileSystem := NewFilesystem()
			if err := fileSystem.Mount(tt.arg.cluster); err != nil {
				t.Errorf("%s failed: %v", tt.name, err)
			}
		})
	}
}

func TestUnMount(t *testing.T) {
	type args struct {
		cluster *v1.Cluster
	}
	tests := []struct {
		name    string
		arg     args
		wantErr bool
	}{
		{
			name: "test unmount",
			arg: args{
				cluster: &v1.Cluster{
					ObjectMeta: v2.ObjectMeta{
						Name: "cluster",
					},
					Spec: v1.ClusterSpec{
						Image: "kuberentes:v1.18.6",
						Masters: v1.Hosts{
							IPList: []string{
								"192.168.56.111",
							},
						},
						Nodes: v1.Hosts{
							IPList: []string{
								"192.168.56.112",
							},
						},
						SSH: v1.SSH{
							User:   "root",
							Passwd: "huaijiahui.com",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileSystem := NewFilesystem()
			if err := fileSystem.UnMount(tt.arg.cluster); err != nil {
				t.Errorf("%s failed: %v", tt.name, err)
			}
		})
	}
}
