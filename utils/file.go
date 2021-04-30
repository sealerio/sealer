package utils

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"gitlab.alibaba-inc.com/seadent/pkg/common"
	"gitlab.alibaba-inc.com/seadent/pkg/logger"
)

// ReadAll read file content
func ReadAll(fileName string) ([]byte, error) {
	// step1：check file exist
	_, err := os.Stat(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
	}
	// step2：open file
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	// step3：read file content
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	//step4：close file
	defer file.Close()
	return content, nil
}

// file ./test/dir/xxx.txt if dir ./test/dir not exist, create it
func MkFileFullPathDir(fileName string) error {
	localDir := filepath.Dir(fileName)
	err := Mkdir(localDir)
	if err != nil {
		return fmt.Errorf("create local dir failed %s %v", localDir, err)
	}
	return nil
}

func Mkdir(dirName string) error {
	return os.MkdirAll(dirName, os.ModePerm)
}

func MkTmpdir() (string, error) {
	tempDir := filepath.Join(os.TempDir(), GenUniqueID(32))
	return tempDir, os.MkdirAll(tempDir, os.ModePerm)
}

func IsFileExist(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func WriteFile(fileName string, content []byte) error {
	dir := filepath.Dir(fileName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, common.FileMode0766); err != nil {
			return err
		}
	}

	if err := ioutil.WriteFile(fileName, content, common.FileMode0766); err != nil {
		return err
	}
	return nil
}

// copy a.txt /var/lib/a.txt
// copy /root/test/abc /tmp/abc
func RecursionCopy(src, dst string) error {
	if IsDir(src) {
		return CopyDir(src, dst)
	}
	_, err := CopySingleFile(src, dst)
	if err != nil {
		return err
	}
	return nil
}

// cp -r /roo/test/* /tmp/abc
func CopyDir(srcPath, dstPath string) error {
	fi, err := ioutil.ReadDir(srcPath)
	if err != nil {
		return err
	}
	for _, f := range fi {
		src := filepath.Join(srcPath, f.Name())
		dst := filepath.Join(dstPath, f.Name())
		if f.IsDir() {
			err = CopyDir(src, dst)
			if err != nil {
				return err
			}
		} else {
			_, err = CopySingleFile(src, dst)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// cp a.txt /tmp/mytest/a.txt
func CopySingleFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	dir := filepath.Dir(dst)
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0766); err != nil {
			return 0, err
		}
	}
	//will over write dst when dst is exist
	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func CleanFile(file *os.File) {
	if file == nil {
		return
	}
	// the following operation won't failed regularly, if failed, log it
	err := file.Close()
	if err != nil {
		logger.Warn(err)
	}
	err = os.Remove(file.Name())
	if err != nil {
		logger.Warn(err)
	}
	return
}

func CleanDir(dir string) (err error) {
	if dir == "" {
		return errors.New("dir name is empty")
	}
	err = os.RemoveAll(dir)
	return
}

func CleanDirs(dirs ...string) (err error) {
	if len(dirs) == 0 {
		return nil
	}
	for _, dir := range dirs {
		err = CleanDir(dir)
		if err != nil {
			return err
		}
	}
	return
}
func CleanFiles(file ...string) error {
	for _, f := range file {
		err := os.RemoveAll(f)
		if err != nil {
			return fmt.Errorf("failed to clean file %s", f)
		}
	}
	return nil
}

func AppendFile(fileName string, content string) error {
	bs, err := ReadAll(fileName)
	if err != nil {
		return errors.Wrapf(err, "read file %s failed", fileName)
	}
	if strings.Contains(string(bs), content) {
		return nil
	}
	err = WriteFile(fileName, []byte(fmt.Sprintf("%s\n%s", bs, content)))
	if err != nil {
		return errors.Wrapf(err, "write file %s failed", fileName)
	}
	return nil
}

func RemoveFileContent(fileName string, content string) error {
	bs, err := ReadAll(fileName)
	if err != nil {
		return errors.Wrapf(err, "read file %s failed", fileName)
	}
	//body := strings.TrimLeft(string(bs), content)
	body := strings.Split(string(bs), content)
	if len(body) != 2 {
		return fmt.Errorf("remove file content failed %s %s", fileName, content)
	}
	err = WriteFile(fileName, []byte(body[0]+body[1]))
	if err != nil {
		return errors.Wrapf(err, "write file %s failed", fileName)
	}
	return nil
}

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func CountDirFiles(dirName string) int {
	if !IsDir(dirName) {
		return 0
	}
	var count int
	err := filepath.Walk(dirName, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		count++
		return nil
	})
	if err != nil {
		logger.Warn("count dir files failed %v", err)
		return 0
	}
	return count
}

func MkDirIfNotExists(dir string) (err error) {
	if _, err = os.Stat(dir); err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(dir, common.FileMode0766)
	}
	//this operation won't fail regularly, so we would logger the err
	if err != nil {
		logger.Error("failed to mkdir, err %s", err)
	}
	return err
}
