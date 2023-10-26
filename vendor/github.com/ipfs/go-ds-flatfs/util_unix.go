//go:build !windows
// +build !windows

package flatfs

import (
	"io/ioutil"
	"os"
)

func tempFileOnce(dir, pattern string) (*os.File, error) {
	return ioutil.TempFile(dir, pattern)
}

func readFileOnce(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}
