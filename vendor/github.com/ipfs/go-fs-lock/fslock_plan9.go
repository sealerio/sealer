package fslock

import "strings"

// Opening an exclusive-use file returns an error.
// The expected error strings are:
//
//  - "open/create -- file is locked" (cwfs, kfs)
//  - "exclusive lock" (fossil)
//  - "exclusive use file already open" (ramfs)
//
// See https://github.com/golang/go/blob/go1.15rc1/src/cmd/go/internal/lockedfile/lockedfile_plan9.go#L16
var lockedErrStrings = [...]string{
	"file is locked",
	"exclusive lock",
	"exclusive use file already open",
}

// isLockedPlan9 return whether an os.OpenFile error indicates that
// a file with the ModeExclusive bit set is already open.
func isLockedPlan9(s string) bool {
	for _, frag := range lockedErrStrings {
		if strings.Contains(s, frag) {
			return true
		}
	}
	return false
}

func lockedByOthers(err error) bool {
	s := err.Error()
	return strings.Contains(s, "Lock Create of") && isLockedPlan9(s)
}
