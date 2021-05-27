package utils

import (
	"reflect"
	"testing"
)

func TestAppendIPList(t *testing.T) {
	type args struct {
		src []string
		dst []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			"test merge ip list",
			args{
				src: []string{"172.16.0.149", "172.16.0.181", "172.16.0.180"},
				dst: []string{"172.16.0.181", "172.16.0.182", "172.16.0.181", "172.16.0.183", "172.16.0.149"},
			},
			[]string{"172.16.0.149", "172.16.0.181", "172.16.0.180", "172.16.0.182", "172.16.0.183"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AppendIPList(tt.args.src, tt.args.dst); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AppendIPList() = %v, want %v", got, tt.want)
			}
		})
	}
}
