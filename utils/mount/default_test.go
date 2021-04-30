package mount

import (
	"testing"
)

func TestDefault_Mount(t *testing.T) {
	type args struct {
		target string
		upper  string
		layers []string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test1 for copy layers to target",
			args: args{
				"../test/mount/default/target",
				"../test/mount/default/upper",
				[]string{"../test/mount/default/layers/layer1"},
			},
			wantErr: false,
		},
		{
			name: "test2 for copy layers to target",
			args: args{
				"../test/mount/default/target",
				"../test/mount/default/upper",
				[]string{"../test/mount/default/layers/layer1", "../test/mount/default/layers/layer2"},
			},
			wantErr: false,
		},
		{
			name: "test3 for copy layers to target",
			args: args{
				"../test/mount/default/target",
				"../test/mount/default/upper",
				[]string{"../test/mount/default/layers/layer1", "../test/mount/default/layers/layer2", "../test/mount/default/layers/layer3"},
			},
			wantErr: false,
		},
		{
			name: "test4 for copy layers to target",
			args: args{
				"../test/mount/default/target",
				"../test/mount/default/upper",
				[]string{"../test/mount/default/layers", "../test/mount/default/layers/layer2",
					"../test/mount/default/layers/layer3", "../test/mount/default/layers/layer4"},
			},
			wantErr: false,
		},
		{
			name: "test5 for copy layers to target",
			args: args{
				"../test/mount/default/target",
				"../test/mount/default/upper",
				[]string{"../test/mount/default/layers/layer1", "../test/mount/default/layers/layer2",
					"../test/mount/default/layers/layer3", "../test/mount/default/layers/layer4"},
			},
			wantErr: false,
		},
		{
			name: "test6 for copy layers to target where target is empty",
			args: args{
				"",
				"../test/mount/default/upper",
				[]string{"../test/mount/default/layers/layer1", "../test/mount/default/layers/layer2"},
			},
			wantErr: true,
		},
		{
			name: "test7 for copy layer file to target",
			args: args{
				"../test/target",
				"../test/mount/default/upper",
				//[]string{"../test/mount/default/layers/layer1/123.txt", "../test/mount/default/layers/layer2/test1.txt"},
				[]string{"../test/mount/default/layers/layer2/test1.txt"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Default{}
			err := d.Mount(tt.args.target, tt.args.upper, tt.args.layers...)
			if (err != nil) != tt.wantErr {
				t.Errorf("err: %s", err)
			}
		})
	}
}
