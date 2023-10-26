// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package goissue34681 allows for files to be opened
// with Windows FILE_SHARE_DELETE flag.

package goissue34681

import (
	"os"
)

// Open is the same as os.Open except it passes FILE_SHARE_DELETE flag
// to Windows CreateFile API.
func Open(name string) (*os.File, error) {
	return open(name)
}

// OpenFile is the same as os.OpenFile except it passes FILE_SHARE_DELETE
// flag to Windows CreateFile API.
func OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return openFile(name, flag, perm)
}
