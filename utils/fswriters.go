package utils

import (
	"os"
)

type atomicFileWriter struct {
	f    *os.File
	path string
	perm os.FileMode
}

func (a *atomicFileWriter) close() (err error) {
	if err = a.f.Sync(); err != nil {
		a.f.Close()
		return err
	}
	if err := a.f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(a.f.Name(), a.perm); err != nil {
		return err
	}
	return os.Rename(a.f.Name(), a.path)
}

func newAtomicFileWriter(path string, perm os.FileMode) (*atomicFileWriter, error) {
	tmpFile, err := MkTmpFile()
	if err != nil {
		return nil, err
	}
	return &atomicFileWriter{f: tmpFile, path: path, perm: perm}, nil
}

func AtomicWriteFile(filepath string, data []byte, perm os.FileMode) (err error) {
	afw, err := newAtomicFileWriter(filepath, perm)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			CleanFile(afw.f)
		}
	}()
	if _, err = afw.f.Write(data); err != nil {
		return
	}
	err = afw.close()
	return
}
