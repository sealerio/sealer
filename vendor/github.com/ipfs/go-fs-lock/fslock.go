package fslock

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	util "github.com/ipfs/go-ipfs-util"
	logging "github.com/ipfs/go-log/v2"
	lock "go4.org/lock"
)

// log is the fsrepo logger
var log = logging.Logger("lock")

// LockedError is returned as the inner error type when the lock is already
// taken.
type LockedError string

func (e LockedError) Error() string {
	return string(e)
}

// Lock creates the lock.
func Lock(confdir, lockFileName string) (io.Closer, error) {
	lockFilePath := filepath.Join(confdir, lockFileName)
	lk, err := lock.Lock(lockFilePath)
	if err != nil {
		switch {
		case lockedByOthers(err):
			return lk, &os.PathError{
				Op:   "lock",
				Path: lockFilePath,
				Err:  LockedError("someone else has the lock"),
			}
		case strings.Contains(err.Error(), "already locked"):
			// we hold the lock ourselves
			return lk, &os.PathError{
				Op:   "lock",
				Path: lockFilePath,
				Err:  LockedError("lock is already held by us"),
			}
		case os.IsPermission(err) || isLockCreatePermFail(err):
			// lock fails on permissions error

			// Using a path error like this ensures that
			// os.IsPermission works on the returned error.
			return lk, &os.PathError{
				Op:   "lock",
				Path: lockFilePath,
				Err:  os.ErrPermission,
			}
		}
	}
	return lk, err
}

// Locked checks if there is a lock already set.
func Locked(confdir, lockFile string) (bool, error) {
	log.Debugf("Checking lock")
	if !util.FileExists(filepath.Join(confdir, lockFile)) {
		log.Debugf("File doesn't exist: %s", filepath.Join(confdir, lockFile))
		return false, nil
	}

	lk, err := Lock(confdir, lockFile)
	if err == nil {
		log.Debugf("No one has a lock")
		lk.Close()
		return false, nil
	}

	log.Debug(err)

	if errors.As(err, new(LockedError)) {
		return true, nil
	}
	return false, err
}

func isLockCreatePermFail(err error) bool {
	s := err.Error()
	return strings.Contains(s, "Lock Create of") && strings.Contains(s, "permission denied")
}
