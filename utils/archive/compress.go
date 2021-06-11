// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package archive

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"syscall"

	"io"

	"os"
	"path/filepath"
	"strings"

	"github.com/alibaba/sealer/common"
)

const compressionBufSize = 32768

type Options struct {
	Compress    bool
	KeepRootDir bool
	ToStream    bool
}

func validatePath(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("dir %s does not exist, err: %s", path, err)
	}
	return nil
}

// TarWithRootDir
// src is the dir or single file to tar
// not contain the dir
// newFolder is a folder for tar file
func TarWithRootDir(paths ...string) (readCloser io.ReadCloser, err error) {
	return compress(paths, Options{Compress: false, KeepRootDir: true})
}

// TarWithoutRootDir function will tar files, but without keeping the original dir
// this is useful when we tar files at the build stage
func TarWithoutRootDir(paths ...string) (readCloser io.ReadCloser, err error) {
	return compress(paths, Options{Compress: false, KeepRootDir: false})
}

func Untar(src io.Reader, dst string) (int64, error) {
	return Decompress(src, dst, Options{Compress: false})
}

// GzipCompress make the tar stream to be gzip stream.
func GzipCompress(in io.Reader) (io.ReadCloser, chan struct{}) {
	compressionDone := make(chan struct{})

	pipeReader, pipeWriter := io.Pipe()
	// Use a bufio.Writer to avoid excessive chunking in HTTP request.
	bufWriter := bufio.NewWriterSize(pipeWriter, compressionBufSize)
	compressor := gzip.NewWriter(bufWriter)

	go func() {
		_, err := io.Copy(compressor, in)
		if err == nil {
			err = compressor.Close()
		}
		if err == nil {
			err = bufWriter.Flush()
		}
		if err != nil {
			// leave the err
			_ = pipeWriter.CloseWithError(err)
		} else {
			pipeWriter.Close()
		}
		close(compressionDone)
	}()

	return pipeReader, compressionDone
}

func compress(paths []string, options Options) (reader io.ReadCloser, err error) {
	if len(paths) == 0 {
		return nil, errors.New("[archive] source must be provided")
	}
	for _, path := range paths {
		err = validatePath(path)
		if err != nil {
			return nil, err
		}
	}

	pr, pw := io.Pipe()
	tw := tar.NewWriter(pw)
	bufWriter := bufio.NewWriterSize(nil, compressionBufSize)
	if options.Compress {
		tw = tar.NewWriter(gzip.NewWriter(pw))
	}
	go func() {
		defer func() {
			tw.Close()
			pw.Close()
		}()

		for _, path := range paths {
			err = writeToTarWriter(path, tw, bufWriter, options)
			if err != nil {
				_ = pw.CloseWithError(err)
			}
		}
	}()

	return pr, nil
}

func writeToTarWriter(path string, tarWriter *tar.Writer, bufWriter *bufio.Writer, options Options) error {
	var newFolder string
	if options.KeepRootDir {
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			newFolder = filepath.Base(path)
		}
	}

	dir := strings.TrimSuffix(path, "/")
	srcPrefix := filepath.ToSlash(dir + "/")
	err := filepath.Walk(dir, func(file string, fi os.FileInfo, err error) error {
		// generate tar header
		header, walkErr := tar.FileInfoHeader(fi, file)
		if walkErr != nil {
			return walkErr
		}
		if file != dir {
			absPath := filepath.ToSlash(file)
			header.Name = filepath.Join(newFolder, strings.TrimPrefix(absPath, srcPrefix))
		} else {
			// do not contain root dir
			if fi.IsDir() {
				return nil
			}
			// for supporting tar single file
			header.Name = filepath.Join(newFolder, filepath.Base(dir))
		}

		walkErr = tarWriter.WriteHeader(header)
		if walkErr != nil {
			return walkErr
		}
		// if not a dir, write file content
		if !fi.IsDir() {
			data, walkErr := os.Open(file)
			if walkErr != nil {
				return walkErr
			}
			defer data.Close()

			bufWriter.Reset(tarWriter)
			defer bufWriter.Reset(nil)

			_, walkErr = io.Copy(bufWriter, data)
			if walkErr != nil {
				return walkErr
			}

			walkErr = bufWriter.Flush()
			if walkErr != nil {
				return walkErr
			}
		}
		return nil
	})

	return err
}

// Decompress this will not change the metadata of original files
func Decompress(src io.Reader, dst string, options Options) (int64, error) {
	// need to set umask to be 000 for current process.
	// there will be some files having higher permission like 777,
	// eventually permission will be set to 755 when umask is 022.
	oldMask := syscall.Umask(0)
	defer syscall.Umask(oldMask)

	err := os.MkdirAll(dst, common.FileMode0755)
	if err != nil {
		return 0, err
	}

	reader := src
	if options.Compress {
		reader, err = gzip.NewReader(src)
		if err != nil {
			return 0, err
		}
	}

	var (
		size int64 = 0
		dirs []*tar.Header
		tr   = tar.NewReader(reader)
	)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		size += header.Size
		// validate name against path traversal
		if !validRelPath(header.Name) {
			return 0, fmt.Errorf("tar contained invalid name error %q", header.Name)
		}

		target := filepath.Join(dst, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err = os.Stat(target); err != nil {
				if err = os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
					return 0, err
				}
				dirs = append(dirs, header)
			}

		case tar.TypeReg:
			err = func() error {
				// regularly won't mkdir, unless add newFolder on compressing
				inErr := os.MkdirAll(filepath.Dir(target), 0755)
				if inErr != nil {
					return inErr
				}

				fileToWrite, inErr := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.FileMode(header.Mode))
				if inErr != nil {
					return inErr
				}

				defer fileToWrite.Close()
				if _, inErr = io.Copy(fileToWrite, tr); inErr != nil {
					return inErr
				}
				// for not changing
				return os.Chtimes(target, header.AccessTime, header.ModTime)
			}()

			if err != nil {
				return 0, err
			}
		}
	}

	for _, h := range dirs {
		path := filepath.Join(dst, h.Name)
		err = os.Chtimes(path, h.AccessTime, h.ModTime)
		if err != nil {
			return 0, err
		}
	}

	return size, nil
}

// check for path traversal and correct forward slashes
func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}
