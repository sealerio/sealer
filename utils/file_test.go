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

package utils

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
			"test from ./test/file/123.txt",
			args{fileName: "./test/file/123.txt"},
			[]byte("123456"),
		},
		{
			"test from ./test/file/abc.txt",
			args{fileName: "./test/file/abc.txt"},
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

type FakeDir struct {
	path  string
	dirs  []FakeDir
	files []FakeFile
}

type FakeFile struct {
	name    string
	content string
}

func makeFakeSourceDir(dir FakeDir) error {
	var (
		err     error
		fi      *os.File
		curRoot = dir.path
	)

	err = os.MkdirAll(curRoot, 0755)
	if err != nil {
		return err
	}
	for _, f := range dir.files {
		fi, err = os.Create(filepath.Join(dir.path, f.name))
		if err != nil {
			return err
		}
		_, err = fi.Write([]byte(f.content))
		if err != nil {
			fi.Close()
			return err
		}
		fi.Close()
	}

	for _, d := range dir.dirs {
		if !strings.HasPrefix(d.path, curRoot) {
			d.path = filepath.Join(curRoot, d.path)
		}
		err = makeFakeSourceDir(d)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestRecursionHardLink(t *testing.T) {
	var (
		err     error
		dstPath = "/tmp/link-test-dst"
	)

	testDir := FakeDir{
		path: "/tmp/link-test",
		dirs: []FakeDir{
			{
				path: "subtest",
				files: []FakeFile{
					{
						name:    "a",
						content: "a",
					},
					{
						name:    "b",
						content: "b",
					},
				},
				dirs: []FakeDir{
					{
						path: "deepSubtest",
						files: []FakeFile{
							{
								name:    "e",
								content: "e",
							},
						},
					},
				},
			},
		},
		files: []FakeFile{
			{
				name:    "c",
				content: "c",
			},
			{
				name:    "d",
				content: "d",
			},
		},
	}

	err = makeFakeSourceDir(testDir)
	defer func() {
		err = os.RemoveAll(testDir.path)
		if err != nil {
			t.Logf("failed to remove all source files, %v", err)
		}
		err = os.RemoveAll(dstPath)
		if err != nil {
			t.Logf("failed to remove all dst files, %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("failed to make fake dir, err: %v", err)
	}
}

func TestAppendFile(t *testing.T) {
	type args struct {
		content  string
		fileName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"add hosts",
			args{
				content:  "127.0.0.1 localhost",
				fileName: "./test/file/abc.txt",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AppendFile(tt.args.fileName, tt.args.content); (err != nil) != tt.wantErr {
				t.Errorf("AppendFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRemoveFileContent(t *testing.T) {
	type args struct {
		fileName string
		content  string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"delete hosts",
			args{
				content:  "127.0.0.1 localhost",
				fileName: "./test/file/abc.txt",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RemoveFileContent(tt.args.fileName, tt.args.content); (err != nil) != tt.wantErr {
				t.Errorf("RemoveFileContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
