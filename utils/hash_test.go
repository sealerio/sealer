package utils

import "testing"

/*func TestDirMD5(t *testing.T) {
	type args struct {
		dirName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"test from ./test/md5",
			args{dirName: "./test/md5"},
			"787fc328e6cd9c254941b3577082b20e",
		},
		{
			"test from ./test/md5/test1.txt",
			args{dirName: "./test/md5/test1.txt"},
			"cef48cf44f5e4f117b789c296b322005",
		},
		{
			"test from ./test/md5/README.md",
			args{dirName: "./test/md5/README.md"},
			"6d853aafea4d6b85d22d09caa1ee5185",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DirMD5(tt.args.dirName); got != tt.want {
				t.Errorf("DirMD5() = %v, want %v", got, tt.want)
			}
		})
	}
}*/

func TestMD5(t *testing.T) {
	type args struct {
		body []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"test md5",
			args{body: []byte("test data")},
			"eb733a00c0c9d336e65691a37ab54293",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MD5(tt.args.body); got != tt.want {
				t.Errorf("MD5() = %v, want %v", got, tt.want)
			}
		})
	}
}
