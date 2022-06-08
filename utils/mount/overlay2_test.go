// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package mount

/*func TestOverlay2_Mount(t *testing.T) {
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
}*/
