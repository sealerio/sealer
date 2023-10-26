package fslock

import (
	"errors"
	"strings"

	"golang.org/x/sys/windows"
)

func lockedByOthers(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) || strings.Contains(err.Error(), "being used by another process")
}
