package util

import (
	"os"
	"strings"
)

func IsFile(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !stat.IsDir()
}

func isFormatFile(path, format string) bool {
	pathSlice := strings.Split(path, ".")
	return pathSlice[len(pathSlice)-1] == format
}

func IsJsonFile(path string) bool {
	return isFormatFile(path, "json")
}

func IsCSVFile(path string) bool {
	return isFormatFile(path, "csv")
}
