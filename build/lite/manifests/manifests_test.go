package manifest

import (
	"reflect"
	"testing"
)

func TestDefaultManifestsRootDir(t *testing.T) {
	type args struct {
		clusterName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"test default manifests root directory",
			args{"my_cluster"},
			"/var/lib/sealer/data/my_cluster/rootfs/manifests",
		},
		{
			"test default manifests root directory 2",
			args{"second_cluster"},
			"/var/lib/sealer/data/second_cluster/rootfs/manifests",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultManifestsRootDir(tt.args.clusterName)
			if got != tt.want {
				t.Errorf("defaultManifestsRootDir() got = %v, want %v", got, tt.want)
				return
			}
		})
	}
}

func TestListImages(t *testing.T) {
	type args struct {
		clusterName string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			"test list manifests images",
			args{"my_cluster"},
			[]string{"k8s.gcr.io/etcd:3.4.13-0", "k8s.gcr.io/kube-apiserver:v1.19.7", "k8s.gcr.io/kube-controller-manager:v1.19.7", "k8s.gcr.io/kube-scheduler:v1.19.7"},
			false,
		},
	}
	manifests, _ := NewManifests()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manifests.ListImages(tt.args.clusterName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListImages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListImages() got = %v, want %v", got, tt.want)
			}
		})
	}
}
