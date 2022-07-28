package buildah

import (
	"github.com/pkg/errors"
	"os"
	"path/filepath"
)

// DiscoverKubefile tries to find a Kubefile within the provided `path`.
func DiscoverKubefile(path string) (foundFile string, err error) {
	// Test for existence of the file
	target, err := os.Stat(path)
	if err != nil {
		return "", errors.Wrap(err, "discovering Kubefile")
	}

	switch mode := target.Mode(); {
	case mode.IsDir():
		// If the path is a real directory, we assume a Kubefile within it
		kubefile := filepath.Join(path, "Kubefile")

		// Test for existence of the Kubefile file
		file, err := os.Stat(kubefile)
		if err != nil {
			return "", errors.Wrap(err, "cannot find Kubefile in context directory")
		}

		// The file exists, now verify the correct mode
		if mode := file.Mode(); mode.IsRegular() {
			foundFile = kubefile
		} else {
			return "", errors.Errorf("assumed Kubefile %q is not a file", kubefile)
		}

	case mode.IsRegular():
		// If the context dir is a file, we assume this as Kubefile
		foundFile = path
	}

	return foundFile, nil
}
