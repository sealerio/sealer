package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

/*
config in Clusterfile:

apiVersion: sealer.aliyun.com/v1alpha1
kind: Config
metadata:
  name: redis-config
spec:
  path: etc/redis-config.yaml
  data: |
       redis-user: root
       redis-passwd: xxx

Dump will dump the config to etc/redis-config.yaml file
*/

type Interface interface {
	// dump Config in Clusterfile to the cluster rootfs disk
	Dump(clusterfile string) error
}

type Dumper struct {
	configs     []v1.Config
	clusterName string
}

func NewConfiguration(clusterName string) Interface {
	return &Dumper{
		clusterName: clusterName,
	}
}

func (c *Dumper) Dump(clusterfile string) error {
	file, err := os.Open(clusterfile)
	if err != nil {
		return fmt.Errorf("failed to dump config %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Warn("failed to dump config close clusterfile failed %v", err)
		}
	}()

	d := yaml.NewYAMLOrJSONDecoder(file, 4096)
	for {
		ext := runtime.RawExtension{}
		if err := d.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		// TODO: This needs to be able to handle object in other encodings and schemas.
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		// ext.Raw
		err := c.DecodeConfig(ext.Raw)
		if err != nil {
			return fmt.Errorf("failed to decode config file %v", err)
		}
	}

	err = c.WriteFiles()
	if err != nil {
		return fmt.Errorf("failed to write config files %v", err)
	}
	return nil
}

func (c *Dumper) WriteFiles() error {
	for _, config := range c.configs {
		err := utils.WriteFile(filepath.Join(common.DefaultTheClusterRootfsDir(c.clusterName), config.Spec.Path), []byte(config.Spec.Data))
		if err != nil {
			return fmt.Errorf("write config file failed %v", err)
		}
	}

	return nil
}

func (c *Dumper) DecodeConfig(Body []byte) error {
	config := v1.Config{}
	err := yaml.Unmarshal(Body, &config)
	if err != nil {
		return fmt.Errorf("decode config failed %v", err)
	}
	if config.Kind == common.CRDConfig {
		c.configs = append(c.configs, config)
	}

	return nil
}
