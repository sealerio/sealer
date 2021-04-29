package mount

import "testing"

func TestOverlay2_Mount(t *testing.T) {
	type args struct {
		target string
		upper  string
		layers []string
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "merged layer files to merged1",
			args: args{
				"../test/mount/overlay2/merged1",
				"../test/mount/overlay2/upper",
				[]string{"../test/mount/overlay2/lower1", "../test/mount/overlay2/lower2", "../test/mount/overlay2/lower3"},
			},
		},
		{
			name: "merged layer files to merged2",
			args: args{
				"../test/mount/overlay2/merged2",
				"../test/mount/overlay2/upper",
				[]string{"../test/mount/overlay2/lower1", "../test/mount/overlay2/lower2", "../test/mount/overlay2/lower3"},
			},
		},
	}
	for _, tt := range tests {
		d := &Overlay2{}
		t.Run(tt.name, func(t *testing.T) {
			if err := d.Mount(tt.args.target, tt.args.upper, tt.args.layers...); err != nil {
				t.Errorf("err %s, %s", tt.name, err)
			} else {
				if err := unmount(tt.args.target, 0); err != nil {
					t.Errorf("err unmount %s, %s", tt.args.target, err)
				}
			}
		})
	}
}
func TestNewMountDriver(t *testing.T) {
	if !supportsOverlay() {
		t.Errorf("mountDriver isn't overlay")
	}
}
