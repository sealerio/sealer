package charts

import (
	"reflect"
	"testing"
)

func TestDefaultChartsRootDir(t *testing.T) {
	type args struct {
		clusterName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"test default charts root directory",
			args{"my_cluster"},
			"/var/lib/sealer/data/my_cluster/rootfs/chars",
		},
		{
			"test default charts root directory 2",
			args{"second_cluster"},
			"/var/lib/sealer/data/second_cluster/rootfs/chars",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultChartsRootDir(tt.args.clusterName)
			if got != tt.want {
				t.Errorf("defaultChartsRootDir() got = %v, want %v", got, tt.want)
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
			"test list chars images",
			args{"my_cluster"},
			[]string{"docker.elastic.co/elasticsearch/elasticsearch:7.13.2", "traefik:2.4.9"},
			false,
		},
	}
	charts, _ := NewChars()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := charts.ListImages(tt.args.clusterName)
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
