// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goissue34681

import (
	"os"
	"syscall"
)

func open(name string) (*os.File, error) {
	return openFile(name, os.O_RDONLY, 0)
}

func openFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	f, err := openFileNolog(name, flag, perm)
	if err != nil {
		return nil, err
	}
	// TODO: deal with f.appendMode
	//f.appendMode = flag&os.O_APPEND != 0

	return f, nil
}

func openFileNolog(name string, flag int, perm os.FileMode) (*os.File, error) {
	if name == "" {
		return nil, &os.PathError{"open", name, syscall.ENOENT}
	}
	f, err := syscallOpen(fixLongPath(name), flag)
	if err != nil {
		return nil, &os.PathError{"open", name, err}
	}
	return os.NewFile(uintptr(f), name), nil
}

// isPathSeparator reports whether c is a directory separator character.
func isPathSeparator(c uint8) bool {
	// NOTE: Windows accept / as path separator.
	return c == '\\' || c == '/'
}

func isAbs(path string) (b bool) {
	v := volumeName(path)
	if v == "" {
		return false
	}
	path = path[len(v):]
	if path == "" {
		return false
	}
	return isPathSeparator(path[0])
}

func volumeName(path string) (v string) {
	if len(path) < 2 {
		return ""
	}
	// with drive letter
	c := path[0]
	if path[1] == ':' &&
		('0' <= c && c <= '9' || 'a' <= c && c <= 'z' ||
			'A' <= c && c <= 'Z') {
		return path[:2]
	}
	// is it UNC
	if l := len(path); l >= 5 && isPathSeparator(path[0]) && isPathSeparator(path[1]) &&
		!isPathSeparator(path[2]) && path[2] != '.' {
		// first, leading `\\` and next shouldn't be `\`. its server name.
		for n := 3; n < l-1; n++ {
			// second, next '\' shouldn't be repeated.
			if isPathSeparator(path[n]) {
				n++
				// third, following something characters. its share name.
				if !isPathSeparator(path[n]) {
					if path[n] == '.' {
						break
					}
					for ; n < l; n++ {
						if isPathSeparator(path[n]) {
							break
						}
					}
					return path[:n]
				}
				break
			}
		}
	}
	return ""
}

// fixLongPath returns the extended-length (\\?\-prefixed) form of
// path when needed, in order to avoid the default 260 character file
// path limit imposed by Windows. If path is not easily converted to
// the extended-length form (for example, if path is a relative path
// or contains .. elements), or is short enough, fixLongPath returns
// path unmodified.
//
// See https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx#maxpath
func fixLongPath(path string) string {
	// Do nothing (and don't allocate) if the path is "short".
	// Empirically (at least on the Windows Server 2013 builder),
	// the kernel is arbitrarily okay with < 248 bytes. That
	// matches what the docs above say:
	// "When using an API to create a directory, the specified
	// path cannot be so long that you cannot append an 8.3 file
	// name (that is, the directory name cannot exceed MAX_PATH
	// minus 12)." Since MAX_PATH is 260, 260 - 12 = 248.
	//
	// The MSDN docs appear to say that a normal path that is 248 bytes long
	// will work; empirically the path must be less then 248 bytes long.
	if len(path) < 248 {
		// Don't fix. (This is how Go 1.7 and earlier worked,
		// not automatically generating the \\?\ form)
		return path
	}

	// The extended form begins with \\?\, as in
	// \\?\c:\windows\foo.txt or \\?\UNC\server\share\foo.txt.
	// The extended form disables evaluation of . and .. path
	// elements and disables the interpretation of / as equivalent
	// to \. The conversion here rewrites / to \ and elides
	// . elements as well as trailing or duplicate separators. For
	// simplicity it avoids the conversion entirely for relative
	// paths or paths containing .. elements. For now,
	// \\server\share paths are not converted to
	// \\?\UNC\server\share paths because the rules for doing so
	// are less well-specified.
	if len(path) >= 2 && path[:2] == `\\` {
		// Don't canonicalize UNC paths.
		return path
	}
	if !isAbs(path) {
		// Relative path
		return path
	}

	const prefix = `\\?`

	pathbuf := make([]byte, len(prefix)+len(path)+len(`\`))
	copy(pathbuf, prefix)
	n := len(path)
	r, w := 0, len(prefix)
	for r < n {
		switch {
		case isPathSeparator(path[r]):
			// empty block
			r++
		case path[r] == '.' && (r+1 == n || isPathSeparator(path[r+1])):
			// /./
			r++
		case r+1 < n && path[r] == '.' && path[r+1] == '.' && (r+2 == n || isPathSeparator(path[r+2])):
			// /../ is currently unhandled
			return path
		default:
			pathbuf[w] = '\\'
			w++
			for ; r < n && !isPathSeparator(path[r]); r++ {
				pathbuf[w] = path[r]
				w++
			}
		}
	}
	// A drive's root directory needs a trailing \
	if w == len(`\\?\c:`) {
		pathbuf[w] = '\\'
		w++
	}
	return string(pathbuf[:w])
}

func syscallOpen(path string, mode int) (fd syscall.Handle, err error) {
	if len(path) == 0 {
		return syscall.InvalidHandle, syscall.ERROR_FILE_NOT_FOUND
	}
	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return syscall.InvalidHandle, err
	}
	var access uint32
	switch mode & (syscall.O_RDONLY | syscall.O_WRONLY | syscall.O_RDWR) {
	case syscall.O_RDONLY:
		access = syscall.GENERIC_READ
	case syscall.O_WRONLY:
		access = syscall.GENERIC_WRITE
	case syscall.O_RDWR:
		access = syscall.GENERIC_READ | syscall.GENERIC_WRITE
	}
	if mode&syscall.O_CREAT != 0 {
		access |= syscall.GENERIC_WRITE
	}
	if mode&syscall.O_APPEND != 0 {
		access &^= syscall.GENERIC_WRITE
		access |= syscall.FILE_APPEND_DATA
	}
	sharemode := uint32(syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE | syscall.FILE_SHARE_DELETE)
	var createmode uint32
	switch {
	case mode&(syscall.O_CREAT|syscall.O_EXCL) == (syscall.O_CREAT | syscall.O_EXCL):
		createmode = syscall.CREATE_NEW
	case mode&(syscall.O_CREAT|syscall.O_TRUNC) == (syscall.O_CREAT | syscall.O_TRUNC):
		createmode = syscall.CREATE_ALWAYS
	case mode&syscall.O_CREAT == syscall.O_CREAT:
		createmode = syscall.OPEN_ALWAYS
	case mode&syscall.O_TRUNC == syscall.O_TRUNC:
		createmode = syscall.TRUNCATE_EXISTING
	default:
		createmode = syscall.OPEN_EXISTING
	}
	h, e := syscall.CreateFile(pathp, access, sharemode, nil, createmode, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	return h, e
}
