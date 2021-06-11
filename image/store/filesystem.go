package store

import (
	iofs "io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/alibaba/sealer/logger"

	"github.com/alibaba/sealer/common"
	pkgutils "github.com/alibaba/sealer/utils"
	"github.com/pkg/errors"

	"github.com/opencontainers/go-digest"
)

const (
	metadataRootDir  = common.DefaultImageMetaRootDir
	layerdataRootDir = common.DefaultLayerDBDir
)

type StoreBackend interface {
	Get(id digest.Digest) ([]byte, error)
	Set(data []byte) (digest.Digest, error)
	Delete(id digest.Digest) error
	List() ([][]byte, error)
	SetMetadata(id digest.Digest, key string, data []byte) error
	GetMetadata(id digest.Digest, key string) ([]byte, error)
	DeleteMetadata(id digest.Digest, key string) error
}

type filesystem struct {
	sync.RWMutex
	root string
}

func NewFSStoreBackend(root string) (StoreBackend, error) {
	var (
		fs  *filesystem
		err error
	)
	fs = &filesystem{
		root: root,
	}
	if err = pkgutils.MkDirIfNotExists(metadataRootDir); err != nil {
		return nil, errors.Errorf("failed to create storage directory, err: %v", err)
	}

	return fs, nil
}

func metadataDir(v interface{}) string {
	switch v.(type) {
	case digest.Digest:
		dgst, _ := v.(digest.Digest)
		return filepath.Join(metadataRootDir, dgst.Hex()+common.YamlSuffix)
	case string:
		filename, _ := v.(string)
		if strings.Contains(filename, common.YamlSuffix) {
			return filepath.Join(metadataRootDir, filename)
		}
		return filepath.Join(metadataRootDir, filename+common.YamlSuffix)
	}

	return ""
}

func layerdataDir(dgst digest.Digest) string {
	return filepath.Join(layerdataRootDir, dgst.Algorithm().String(), dgst.Hex())
}

func (fs *filesystem) Get(id digest.Digest) ([]byte, error) {
	var (
		metadata []byte
		err      error
	)
	fs.RLock()
	defer fs.RUnlock()

	//we do not use the functions in pkgutils because the validation steps
	//in its function is redundant in this situation
	metadata, err = ioutil.ReadFile(metadataDir(id))
	if err != nil {
		return nil, errors.Errorf("failed to read image %s's metadata, err: %v", id, err)
	}

	if digest.FromBytes(metadata) != id {
		return nil, errors.Errorf("failed to verify image %s's hash value", id)
	}

	return metadata, nil
}

func (fs *filesystem) Set(data []byte) (digest.Digest, error) {
	var (
		dgst digest.Digest
		err  error
	)
	fs.Lock()
	defer fs.Unlock()

	if len(data) == 0 {
		return "", errors.Errorf("invalid empty data")
	}

	dgst = digest.FromBytes(data)
	if err = ioutil.WriteFile(metadataDir(dgst), data, common.FileMode0644); err != nil {
		return "", errors.Errorf("failed to write image %s's metadata, err: %v", dgst, err)
	}

	return dgst, nil
}

func (fs *filesystem) Delete(dgst digest.Digest) error {
	var (
		err error
	)
	fs.Lock()
	defer fs.Unlock()

	if err = os.RemoveAll(metadataDir(dgst)); err != nil {
		return errors.Errorf("failed to delete image %s's metadata, err: %v", err)
	}

	return nil
}

func (fs *filesystem) List() ([][]byte, error) {
	var (
		configs   [][]byte
		err       error
		fileInfos []iofs.FileInfo
	)

	fileInfos, err = ioutil.ReadDir(metadataRootDir)
	if err != nil {
		return nil, errors.Errorf("failed to open metadata directory %s, err: %v",
			metadataRootDir, err)
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			continue
		}

		if strings.Contains(fileInfo.Name(), common.YamlSuffix) {
			config, err := ioutil.ReadFile(metadataDir(fileInfo.Name()))
			if err != nil {
				logger.Error("failed to read file %v, err: %v", fileInfo.Name(), err)
			}
			configs = append(configs, config)
		}
	}

	return configs, nil
}

func (fs *filesystem) SetMetadata(id digest.Digest, key string, data []byte) error {
	fs.Lock()
	defer fs.Unlock()

	baseDir := layerdataDir(id)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(baseDir, key), data, 0755)
}

func (fs *filesystem) GetMetadata(id digest.Digest, key string) ([]byte, error) {
	fs.Lock()
	defer fs.Unlock()

	bytes, err := ioutil.ReadFile(filepath.Join(layerdataDir(id), key))
	if err != nil {
		return nil, errors.Errorf("failed to read metadata, err: %v", err)
	}

	return bytes, nil
}

func (fs *filesystem) DeleteMetadata(id digest.Digest, key string) error {
	fs.Lock()
	defer fs.Unlock()

	return os.RemoveAll(filepath.Join(layerdataDir(id), key))
}
