// Package flatfs is a Datastore implementation that stores all
// objects in a two-level directory structure in the local file
// system, regardless of the hierarchy of the keys.
package flatfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
)

func UpgradeV0toV1(path string, prefixLen int) error {
	fun := Prefix(prefixLen)
	err := WriteShardFunc(path, fun)
	if err != nil {
		return err
	}
	err = WriteReadme(path, fun)
	if err != nil {
		return err
	}
	return nil
}

func DowngradeV1toV0(path string) error {
	fun, err := ReadShardFunc(path)
	if err != nil {
		return err
	} else if fun.funName != "prefix" {
		return fmt.Errorf("%s: can only downgrade datastore that use the 'prefix' sharding function", path)
	}

	err = os.Remove(filepath.Join(path, SHARDING_FN))
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(path, README_FN))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func Move(oldPath string, newPath string, out io.Writer) error {
	oldDS, err := Open(oldPath, false)
	if err != nil {
		return fmt.Errorf("%s: %v", oldPath, err)
	}
	oldDS.deactivate()
	newDS, err := Open(newPath, false)
	if err != nil {
		return fmt.Errorf("%s: %v", newPath, err)
	}
	newDS.deactivate()

	res, err := oldDS.Query(context.Background(), query.Query{KeysOnly: true})
	if err != nil {
		return err
	}

	if out != nil {
		fmt.Fprintf(out, "Moving Keys...\n")
	}

	// first move the keys
	count := 0
	for {
		e, ok := res.NextSync()
		if !ok {
			break
		}
		if e.Error != nil {
			res.Close()
			return e.Error
		}

		err := moveKey(oldDS, newDS, datastore.RawKey(e.Key))
		if err != nil {
			res.Close()
			return err
		}

		count++
		if out != nil && count%10 == 0 {
			fmt.Fprintf(out, "\r%d keys so far", count)
		}
	}
	res.Close()

	if out != nil {
		fmt.Fprintf(out, "\nCleaning Up...\n")
	}

	// now walk the old top-level directory
	dir, err := os.Open(oldDS.path)
	if err != nil {
		return err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, fn := range names {
		if fn == "." || fn == ".." {
			continue
		}
		oldPath := filepath.Join(oldDS.path, fn)
		inf, err := os.Stat(oldPath)
		if err != nil {
			return err
		}
		if inf.IsDir() {
			indir, err := os.Open(oldPath)
			if err != nil {
				return err
			}

			names, err := indir.Readdirnames(-1)
			indir.Close()
			if err != nil {
				return err
			}

			for _, n := range names {
				p := filepath.Join(oldPath, n)
				// part of unfinished write transaction
				// remove it
				if strings.HasPrefix(n, "put-") {
					err := os.Remove(p)
					if err != nil {
						return err
					}
				} else {
					return errors.New("unknown file in flatfs: " + p)
				}
			}

			err = os.Remove(oldPath)
			if err != nil {
				return err
			}
		} else if fn == SHARDING_FN || fn == README_FN {
			// generated file so just remove it
			err := os.Remove(oldPath)
			if err != nil {
				return err
			}
		} else {
			// else we found something unexpected, so to be safe just move it
			log.Warnw("found unexpected file in datastore directory, moving anyways", "file", fn)
			newPath := filepath.Join(newDS.path, fn)
			err := rename(oldPath, newPath)
			if err != nil {
				return err
			}
		}
	}

	if out != nil {
		fmt.Fprintf(out, "All Done.\n")
	}

	return nil
}

func moveKey(oldDS *Datastore, newDS *Datastore, key datastore.Key) error {
	_, oldPath := oldDS.encode(key)
	dir, newPath := newDS.encode(key)
	err := os.Mkdir(dir, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	err = rename(oldPath, newPath)
	if err != nil {
		return err
	}
	return nil
}
