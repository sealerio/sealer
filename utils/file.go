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

package utils

import (
	"archive/tar"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/kubelet/config/v1beta1"

	"github.com/alibaba/sealer/pkg/runtime/kubeadm_types/v1beta2"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kube-proxy/config/v1alpha1"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

func IsExist(fileName string) bool {
	if _, err := os.Stat(fileName); err != nil {
		return os.IsExist(err)
	}
	return true
}

func RemoveDuplicate(list []string) []string {
	var result []string
	flagMap := map[string]struct{}{}
	for _, v := range list {
		if _, ok := flagMap[v]; !ok {
			flagMap[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

func ReadLines(fileName string) ([]string, error) {
	var lines []string
	if !IsExist(fileName) {
		return nil, errors.New("no such file")
	}
	file, err := os.Open(filepath.Clean(fileName))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	br := bufio.NewReader(file)
	for {
		line, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		lines = append(lines, string(line))
	}
	return lines, nil
}

// ReadAll read file content
func ReadAll(fileName string) ([]byte, error) {
	// step1：check file exist
	if !IsExist(fileName) {
		return nil, errors.New("no such file")
	}
	// step2：open file
	file, err := os.Open(filepath.Clean(fileName))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// step3：read file content
	content, err := ioutil.ReadFile(filepath.Clean(fileName))
	if err != nil {
		return nil, err
	}

	return content, nil
}

// file ./test/dir/xxx.txt if dir ./test/dir not exist, create it
func MkFileFullPathDir(fileName string) error {
	localDir := filepath.Dir(fileName)
	if err := Mkdir(localDir); err != nil {
		return fmt.Errorf("failed to create local dir %s: %v", localDir, err)
	}
	return nil
}

func Mkdir(dirName string) error {
	return os.MkdirAll(dirName, os.ModePerm)
}

func MkDirs(dirs ...string) error {
	if len(dirs) == 0 {
		return nil
	}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create %s, %v", dir, err)
		}
	}
	return nil
}

func MkTmpdir() (string, error) {
	tempDir, err := ioutil.TempDir(common.DefaultTmpDir, ".DTmp-")
	if err != nil {
		return "", err
	}
	return tempDir, os.MkdirAll(tempDir, os.ModePerm)
}

func MkTmpFile(path string) (*os.File, error) {
	return ioutil.TempFile(path, ".FTmp-")
}

func IsFileExist(filename string) bool {
	_, err := os.Stat(filename)
	if !os.IsNotExist(err) {
		if err == nil {
			return true
		}
		logger.Warn(err)
	}
	return false
}

func WriteFile(fileName string, content []byte) error {
	dir := filepath.Dir(fileName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, common.FileMode0755); err != nil {
			return err
		}
	}

	if err := AtomicWriteFile(fileName, content, common.FileMode0644); err != nil {
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

	err := os.MkdirAll(filepath.Dir(dst), 0700|0055)
	if err != nil {
		return fmt.Errorf("failed to mkdir for recursion copy, err: %v", err)
	}

	_, err = CopySingleFile(src, dst)
	return err
}

func RecursionHardLink(src, dst string) error {
	if !IsDir(src) {
		return os.Link(src, dst)
	}
	fhs := []*tar.Header{}
	err := RecursionHardLinkDir(src, dst, &fhs)
	if err != nil {
		return fmt.Errorf("failed to recursion hard link dir %s, err: %s", src, err)
	}

	for _, h := range fhs {
		err = os.Chtimes(h.Name, h.AccessTime, h.ModTime)
		if err != nil {
			return fmt.Errorf("failed to chtimes for %s, err: %v", h.Name, err)
		}

		err = os.Chmod(h.Name, os.FileMode(h.Mode))
		if err != nil {
			return fmt.Errorf("failed to chmod for %s, err: %v", h.Name, err)
		}
	}
	return nil
}

func RecursionHardLinkDir(src, dst string, modTimes *[]*tar.Header) error {
	if modTimes == nil {
		return fmt.Errorf("modTimes should be init")
	}

	fis, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	// TODO maybe mk follow the src file
	err = os.MkdirAll(dst, common.FileMode0755)
	if err != nil {
		return err
	}

	for _, f := range fis {
		var (
			srcPath = filepath.Join(src, f.Name())
			dstPath = filepath.Join(dst, f.Name())
		)
		if f.IsDir() {
			err = RecursionHardLinkDir(srcPath, dstPath, modTimes)
			if err != nil {
				return err
			}

			var fh *tar.Header
			fh, err = tar.FileInfoHeader(f, src)
			if err != nil {
				return err
			}
			fh.Name = dstPath
			*modTimes = append(*modTimes, fh)
		} else {
			err = os.Link(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// cp -r /roo/test/* /tmp/abc
func CopyDir(srcPath, dstPath string) error {
	err := os.MkdirAll(dstPath, 0700|0055)
	if err != nil {
		return err
	}

	opaque, err := Lgetxattr(srcPath, "trusted.overlay.opaque")
	if err != nil {
		logger.Debug("failed to get trusted.overlay.opaque. err: %v", err)
	}

	if len(opaque) == 1 && opaque[0] == 'y' {
		err = unix.Setxattr(dstPath, "trusted.overlay.opaque", []byte{'y'}, 0)
		if err != nil {
			return fmt.Errorf("failed to set trusted.overlay.opaque, err: %v", err)
		}
	}

	fis, err := ioutil.ReadDir(srcPath)
	if err != nil {
		return err
	}
	for _, f := range fis {
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

	header, err := tar.FileInfoHeader(sourceFileStat, src)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info header for %s, err: %v", src, err)
	}
	if sourceFileStat.Mode()&os.ModeCharDevice != 0 && header.Devminor == 0 && header.Devmajor == 0 {
		err = unix.Mknod(dst, unix.S_IFCHR, 0)
		if err != nil {
			return 0, err
		}
		return 0, os.Chown(dst, header.Uid, header.Gid)
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(filepath.Clean(src))
	if err != nil {
		return 0, err
	}
	defer source.Close()
	//will overwrite dst when dst is existed
	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	err = destination.Chmod(sourceFileStat.Mode())
	if err != nil {
		return 0, err
	}

	err = os.Chown(dst, header.Uid, header.Gid)
	if err != nil {
		return 0, err
	}
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func CleanFile(file *os.File) {
	if file == nil {
		return
	}
	// the following operation won't failed regularly, if failed, log it
	err := file.Close()
	if err != nil && err != os.ErrClosed {
		logger.Warn(err)
	}
	err = os.Remove(file.Name())
	if err != nil {
		logger.Warn(err)
	}
}

func CleanDir(dir string) {
	if dir == "" {
		logger.Error("clean dir path is empty")
	}

	if err := os.RemoveAll(dir); err != nil {
		logger.Warn("failed to remove dir %s ", dir)
	}
}

func CleanDirs(dirs ...string) {
	if len(dirs) == 0 {
		return
	}
	for _, dir := range dirs {
		CleanDir(dir)
	}
}
func CleanFiles(file ...string) error {
	for _, f := range file {
		err := os.RemoveAll(f)
		if err != nil {
			return fmt.Errorf("failed to clean file %s, %v", f, err)
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

func IsFileContent(fileName string, content string) bool {
	bs, err := ReadAll(fileName)
	if err != nil {
		logger.Error(err)
		return false
	}
	return len(strings.Split(string(bs), content)) == 2
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

func GetFileSize(path string) (size int64, err error) {
	_, err = os.Stat(path)
	if err != nil {
		return
	}
	err = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func GetFilesSize(paths []string) (int64, error) {
	var size int64
	for i := range paths {
		s, err := GetFileSize(paths[i])
		if err != nil {
			return 0, err
		}
		size += s
	}
	return size, nil
}

func DecodeCluster(filepath string) (clusters *v1.Cluster, err error) {
	decodeClusters, err := DecodeCRDFromFile(filepath, common.Cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cluster from %s, %v", filepath, err)
	}
	clusters = decodeClusters.(*v1.Cluster)
	return
}

func DecodeConfigs(filePath string) (configs []v1.Config, err error) {
	decodeConfigs, err := decodeCRDs(filePath, common.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config from %s, %v", filePath, err)
	}
	configs = decodeConfigs.([]v1.Config)
	return
}

func DecodePlugins(filepath string) (plugins []v1.Plugin, err error) {
	decodePlugins, err := decodeCRDs(filepath, common.Plugin)
	if err != nil {
		return nil, fmt.Errorf("failed to decode plugin from %s, %v", filepath, err)
	}
	plugins = decodePlugins.([]v1.Plugin)
	return
}

func decodeCRDs(filePath string, kind string) (out interface{}, err error) {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to dump config %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Warn("failed to dump config close clusterfile failed %v", err)
		}
	}()
	var (
		i       interface{}
		configs []v1.Config
		plugins []v1.Plugin
	)

	d := yaml.NewYAMLOrJSONDecoder(file, 4096)

	for {
		ext := runtime.RawExtension{}
		if err := d.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		// TODO: This needs to be able to handle object in other encodings and schemas.
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		// ext.Raw
		switch kind {
		case common.Config:
			config := v1.Config{}
			err := yaml.Unmarshal(ext.Raw, &config)
			if err != nil {
				return nil, fmt.Errorf("decode config failed %v", err)
			}
			if config.Kind == common.Config {
				configs = append(configs, config)
			}
			i = configs
		case common.Plugin:
			plugin := v1.Plugin{}
			err := yaml.Unmarshal(ext.Raw, &plugin)
			if err != nil {
				return nil, fmt.Errorf("decode plugin failed %v", err)
			}
			if plugin.Kind == common.Plugin {
				plugins = append(plugins, plugin)
			}
			i = plugins
		}
	}

	return i, nil
}

func DecodeCRDFromFile(filePath string, kind string) (interface{}, error) {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to dump config %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Warn("failed to dump config close clusterfile failed %v", err)
		}
	}()
	return DecodeCRDFromReader(file, kind)
}

func DecodeCRDFromReader(r io.Reader, kind string) (interface{}, error) {
	d := yaml.NewYAMLOrJSONDecoder(r, 4096)

	for {
		ext := runtime.RawExtension{}
		if err := d.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		// TODO: This needs to be able to handle object in other encodings and schemas.
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		metaType := metav1.TypeMeta{}
		err := yaml.Unmarshal(ext.Raw, &metaType)
		if err != nil {
			return nil, fmt.Errorf("decode cluster failed %v", err)
		}
		// ext.Raw
		if metaType.Kind == kind {
			return TypeConversion(ext.Raw, kind)
		}
	}
	return nil, nil
}

func DecodeCRDFromString(config string, kind string) (interface{}, error) {
	return DecodeCRDFromReader(strings.NewReader(config), kind)
}

func TypeConversion(raw []byte, kind string) (i interface{}, err error) {
	i = typeConversion(kind)
	if i == nil {
		return nil, fmt.Errorf("not found type %s from %s", kind, string(raw))
	}
	return i, yaml.Unmarshal(raw, i)
}

func typeConversion(kind string) interface{} {
	switch kind {
	case common.Cluster:
		return &v1.Cluster{}
	case common.KubeConfig:
		return &v2.KubeConfig{}
	case common.InitConfiguration:
		return &v1beta2.InitConfiguration{}
	case common.JoinConfiguration:
		return &v1beta2.JoinConfiguration{}
	case common.ClusterConfiguration:
		return &v1beta2.ClusterConfiguration{}
	case common.KubeletConfiguration:
		return &v1beta1.KubeletConfiguration{}
	case common.KubeProxyConfiguration:
		return &v1alpha1.KubeProxyConfiguration{}
	}
	return nil
}
