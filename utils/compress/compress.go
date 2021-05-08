package compress

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/pkg/system"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/utils"
)

// src is the dir or single file to tar
// not contain the dir
// newFolder is a folder for tar file
func Compress(src, newFolder string, existingFile *os.File) (file *os.File, err error) {
	if len(src) == 0 {
		return nil, errors.New("[compress] source must be provided")
	}

	if !filepath.IsAbs(src) {
		return nil, errors.New("src should be absolute path")
	}

	_, err = os.Stat(src)
	if err != nil {
		return
	}

	//use existing file
	file = existingFile
	if file == nil {
		file, err = ioutil.TempFile("/tmp", "seadent_compress")
	}

	if err != nil {
		return nil, errors.New("create tmp compress file failed")
	}

	defer func() {
		if err != nil {
			utils.CleanFile(file)
		}
	}()

	zr := gzip.NewWriter(file)
	tw := tar.NewWriter(zr)
	defer func() {
		_ = tw.Close()
		_ = zr.Close()
	}()

	src = strings.TrimSuffix(src, "/")
	srcPrefix := filepath.ToSlash(src + "/")
	err = filepath.Walk(src, func(file string, fi os.FileInfo, funcErr error) error {
		// generate tar header
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}
		if file != src {
			absPath := filepath.ToSlash(file)
			header.Name = filepath.Join(newFolder, strings.TrimPrefix(absPath, srcPrefix))
		} else {
			// do not contain root dir
			if fi.IsDir() {
				return nil
			}
			// for supporting tar single file
			header.Name = filepath.Join(newFolder, filepath.Base(src))
		}

		// write header
		if err = tw.WriteHeader(header); err != nil {
			return err
		}
		// if not a dir, write file content
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			if _, err = io.Copy(tw, data); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return file, nil
}

// example: dir:/var/lib/etcd target:/home/etcd.tar.gz
// this func will keep original dir etcd
func Dir(dir, target string) (err error) {
	if dir == "" || target == "" {
		return errors.New("dir or target should be provided")
	}

	if !filepath.IsAbs(dir) || !filepath.IsAbs(target) {
		return errors.New("dir and target should be absolute path")
	}

	target = strings.TrimSuffix(target, "/")
	tarDir := filepath.Dir(target)
	if err = os.MkdirAll(tarDir, common.FileMode0755); err != nil {
		return err
	}

	var file *os.File
	if file, err = os.OpenFile(target, os.O_RDWR|os.O_TRUNC|os.O_CREATE, common.FileMode0755); err != nil {
		return err
	}
	defer file.Close()

	dir = strings.TrimSuffix(dir, "/")
	originDir := filepath.Base(dir)
	// the return file will point to the file above, which will be close in defer, so ignore it
	if _, err = Compress(dir, originDir, file); err != nil {
		return err
	}
	return nil
}

// this uncompress will not change the metadata of original files
func Uncompress(src io.Reader, dst string) error {
	// need to set umask to be 000 for current process.
	// there will be some files having higher permission like 777,
	// eventually permission will be set to 755 when umask is 022.
	_, err := system.Umask(0)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dst, common.FileMode0755)
	if err != nil {
		return err
	}

	zr, err := gzip.NewReader(src)
	if err != nil {
		return err
	}

	tr := tar.NewReader(zr)
	type DirStruct struct {
		header     *tar.Header
		dir        string
		next, prev *DirStruct
	}

	prefixes := make(map[string]*DirStruct)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// validate name against path traversal
		if !validRelPath(header.Name) {
			return fmt.Errorf("tar contained invalid name error %q", header.Name)
		}

		target := filepath.Join(dst, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err = os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
					return err
				}

				// building a double linked list
				prefix := filepath.Dir(target)
				prev := prefixes[prefix]
				//an root dir
				if prev == nil {
					prefixes[target] = &DirStruct{header: header, dir: target, next: nil, prev: nil}
				} else {
					newHead := &DirStruct{header: header, dir: target, next: nil, prev: prev}
					prev.next = newHead
					prefixes[target] = newHead
				}
			}

		case tar.TypeReg:
			err = func() error {
				fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.FileMode(header.Mode))
				if err != nil {
					return err
				}

				defer fileToWrite.Close()
				if _, err := io.Copy(fileToWrite, tr); err != nil {
					return err
				}
				// for not changing
				if err = os.Chtimes(target, header.AccessTime, header.ModTime); err != nil {
					return err
				}
				return nil
			}()

			if err != nil {
				return err
			}
		}
	}

	for _, v := range prefixes {
		// for taking the last one
		if v.next != nil {
			continue
		}

		// every change in dir, will change the metadata of that dir
		// change times from the last one
		// do this is for not changing metadata of parent dir
		for dirStr := v; dirStr != nil; dirStr = dirStr.prev {
			if err = os.Chtimes(dirStr.dir, dirStr.header.AccessTime, dirStr.header.ModTime); err != nil {
				return err
			}
		}
	}

	return nil
}

// check for path traversal and correct forward slashes
func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}
