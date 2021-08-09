package plugin

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
)

/*
config in PluginConfig:

apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: SHELL
spec:
  action: PostInstall
  on: role=master
  data: |
    kubectl taint nodes node-role.kubernetes.io/master=:NoSchedule

Dump will dump the config to etc/redis-config.yaml file
*/

type ConfigInterface interface {
	// dump Config in Clusterfile to the cluster rootfs disk
	Dump(clusterfile string) error
	GetPhasePlugin(phase Phase) []v1.Plugin
}

type DumperPlugin struct {
	plugin      []v1.Plugin
	clusterName string
}

func (c *DumperPlugin) GetPhasePlugin(phase Phase) []v1.Plugin {
	configs := make([]v1.Plugin, 0)
	for _, config := range c.plugin {
		on := Phase(config.Spec.On[5:])
		if on == phase {
			configs = append(configs, config)
		}
	}
	return configs
}

func Config(clusterName string) *DumperPlugin {
	return &DumperPlugin{
		clusterName: clusterName,
	}
}

func (c *DumperPlugin) Dump(clusterfile string) error {
	if clusterfile == "" {
		logger.Debug("clusterfile is empty!")
		return nil
	}
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

func (c *DumperPlugin) WriteFiles() error {
	for _, config := range c.plugin {

		err := utils.WriteFile(filepath.Join(common.DefaultTheClusterRootfsPluginDir(c.clusterName), config.ObjectMeta.Name), []byte(config.Spec.Data))
		if err != nil {
			return fmt.Errorf("write config fileed %v", err)
		}
	}

	return nil
}

func (c *DumperPlugin) DecodeConfig(Body []byte) error {
	config := v1.Plugin{}
	err := yaml.Unmarshal(Body, &config)
	if err != nil {
		return fmt.Errorf("decode config failed %v", err)
	}
	if config.Kind == common.CRDConfig {
		c.plugin = append(c.plugin, config)
	}
	return nil
}
