package utils

import (
	"reflect"
	"testing"
)

func TestReadAll(t *testing.T) {
	type args struct {
		fileName string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		// TODO: Add test cases.
		{
			"test from ./test/123.txt",
			args{fileName: "./test/123.txt"},
			[]byte("123456"),
		},
		{
			"test from ./test/abc.txt",
			args{fileName: "./test/abc.txt"},
			[]byte("a\r\nb\r\nc\r\nd"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := ReadAll(tt.args.fileName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMkdir(t *testing.T) {
	type args struct {
		dirName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"test crete dri",
			args{dirName: "./test/deep/dir/test"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Mkdir(tt.args.dirName); (err != nil) != tt.wantErr {
				t.Errorf("Mkdir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCountDirFiles(t *testing.T) {
	type args struct {
		dirName string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			"count dir files",
			args{"."},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CountDirFiles(tt.args.dirName); got < tt.want {
				t.Errorf("CountDirFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyDir(t *testing.T) {
	type args struct {
		src, dst string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"test copyDir when src is dir",
			args{src: "/root/test", dst: "/tmp"},
			false,
		},
		{
			"test copyDir when src is not exist",
			args{src: "/root/Notexist", dst: "/tmp"},
			true,
		},
		{
			"test copyDir when dst is not exist",
			args{src: "/root/test", dst: "/tmp"},
			false,
		},
		{
			"test copyDir when dst is not dir",
			args{src: "/root/test", dst: "/tmp"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CopyDir(tt.args.src, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("CopyDir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCopySingleFile(t *testing.T) {
	type args struct {
		src, dst string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"test copy single file when src is dir",
			args{src: "/root", dst: "/tmp"},
			true,
		},
		{
			"test copy single file when src is not regular file, seems like link file",
			args{src: "/root/link", dst: "/tmp"},
			true,
		},
		{
			"test copy single file when src is not exist",
			args{src: "/root/Notexist", dst: "/tmp"},
			true,
		},

		{
			"test copy single file dst exist",
			args{src: "/root/test", dst: "/tmp"},
			false,
		},
		{
			"test copy single file dst file path not exist",
			args{src: "/root/test", dst: "/tmp/abc"},
			false,
		},
		{
			"test copy single file when dst file is exist",
			args{src: "/root/test", dst: "/tmp/test"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := CopySingleFile(tt.args.src, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("CopySingleFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
