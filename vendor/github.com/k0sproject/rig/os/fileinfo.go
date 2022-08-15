package os

import (
	"io/fs"
	"time"
)

// FileInfo implements fs.FileInfo
type FileInfo struct {
	FName    string
	FSize    int64
	FMode    fs.FileMode
	FModTime time.Time
	FIsDir   bool
}

func (f *FileInfo) Name() string {
	return f.FName
}

func (f *FileInfo) Size() int64 {
	return f.FSize
}

func (f *FileInfo) Mode() fs.FileMode {
	return f.FMode
}

func (f *FileInfo) ModTime() time.Time {
	return f.FModTime
}

func (f *FileInfo) IsDir() bool {
	return f.FIsDir
}
