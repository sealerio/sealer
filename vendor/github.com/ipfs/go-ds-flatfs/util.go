package flatfs

import (
	"io"
	"os"
	"time"
)

// From: http://stackoverflow.com/questions/30697324/how-to-check-if-directory-on-path-is-empty
func DirIsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func readFile(filename string) (data []byte, err error) {
	// Fallback retry for temporary error.
	for i := 0; i < RetryAttempts; i++ {
		data, err = readFileOnce(filename)
		if err == nil || !isTooManyFDError(err) {
			break
		}
		time.Sleep(time.Duration(i+1) * RetryDelay)
	}
	return data, err
}

func tempFile(dir, pattern string) (fi *os.File, err error) {
	for i := 0; i < RetryAttempts; i++ {
		fi, err = tempFileOnce(dir, pattern)
		if err == nil || !isTooManyFDError(err) {
			break
		}
		time.Sleep(time.Duration(i+1) * RetryDelay)
	}
	return fi, err
}
