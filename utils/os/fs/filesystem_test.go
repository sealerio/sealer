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

package fs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			if err := NewFilesystem().MkdirAll(tt.args.dirName); (err != nil) != tt.wantErr {
				t.Errorf("Mkdir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRenameDir(t *testing.T) {
	oldPath := "/tmp/mytest"
	newPath := "/tmp/abc/mytest"
	defer os.RemoveAll(newPath)

	err := FS.MkdirAll(oldPath)
	if err != nil {
		t.Fatalf("TempDir %s: %v", t.Name(), err)
	}

	filename := "tmp-file"
	data := []byte("i am a tmp file\n")
	if err := os.WriteFile(filepath.Join(oldPath, filename), data, 0644); err != nil {
		t.Fatalf("WriteFile %s: %v", filename, err)
	}

	type args struct {
		filename    string
		fileContent []byte
		newPath     string
		oldPath     string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"test rename dri",
			args{
				filename:    filename,
				fileContent: data,
				oldPath:     oldPath,
				newPath:     newPath},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := FS.Rename(tt.args.oldPath, tt.args.newPath); (err != nil) != tt.wantErr {
				t.Errorf("Rename() error = %v, wantErr %v", err, tt.wantErr)
			}

			content, err := os.ReadFile(filepath.Join(tt.args.newPath, filename))
			assert.NoErrorf(t, err, "failed to load file content form new path")
			assert.Equal(t, tt.args.fileContent, content)
		})
	}
}

/*func TestCopyDir(t *testing.T) {
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
			if err := NewFilesystem().CopyDir(tt.args.src, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("CopyDir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
*/

/*func TestCopySingleFile(t *testing.T) {
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
			if _, err := NewFilesystem().CopyFile(tt.args.src, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("CopySingleFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}*/

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
