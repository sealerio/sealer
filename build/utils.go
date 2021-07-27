package build

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/alibaba/sealer/client"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image"
	infraUtils "github.com/alibaba/sealer/infra/utils"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/runtime"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/opencontainers/go-digest"

	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

// GetClusterFile from user build context or from base image
func GetRawClusterFile(im *v1.Image) string {
	if im.Spec.Layers[0].Value == common.ImageScratch {
		data, err := ioutil.ReadFile(filepath.Join("etc", common.DefaultClusterFileName))
		if err != nil {
			return ""
		}
		return string(data)
	}
	// find cluster file from context
	if clusterFile := getClusterFileFromContext(im); clusterFile != nil {
		logger.Info("get cluster file from context success!")
		return string(clusterFile)
	}
	// find cluster file from base image
	clusterFile := image.GetClusterFileFromImage(im.Spec.Layers[0].Value)
	if clusterFile != "" {
		logger.Info("get cluster file from base image success!")
		return clusterFile
	}
	return ""
}

func getClusterFileFromContext(image *v1.Image) []byte {
	for i := range image.Spec.Layers {
		layer := image.Spec.Layers[i]
		if layer.Type == common.COPYCOMMAND && strings.Fields(layer.Value)[0] == common.DefaultClusterFileName {
			if clusterFile, _ := utils.ReadAll(strings.Fields(layer.Value)[0]); clusterFile != nil {
				return clusterFile
			}
		}
	}
	return nil
}

// used in build stage, where the image still has from layer
func getBaseLayersPath(layers []v1.Layer) (res []string) {
	for _, layer := range layers {
		if layer.ID != "" {
			res = append(res, filepath.Join(common.DefaultLayerDir, layer.ID.Hex()))
		}
	}
	return res
}

func generateImageID(image v1.Image) (string, error) {
	imageBytes, err := yaml.Marshal(image)
	if err != nil {
		return "", err
	}
	imageID := digest.FromBytes(imageBytes).Hex()
	return imageID, nil
}

func setClusterFileToImage(image *v1.Image) {
	clusterFileData := GetRawClusterFile(image)

	if image.Annotations == nil {
		image.Annotations = make(map[string]string)
	}
	image.Annotations[common.ImageAnnotationForClusterfile] = clusterFileData
}

func GetRegistryBindDir() string {
	// check is docker running runtime.RegistryName
	// check bind dir
	var registryName = runtime.RegistryName
	var registryDest = runtime.RegistryBindDest
	ctx := context.Background()
	cli, err := client.NewDockerClient()
	if err != nil {
		return ""
	}
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})

	if err != nil {
		return ""
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if strings.Contains(name, registryName) {
				for _, m := range c.Mounts {
					if m.Type == mount.TypeBind && m.Destination == registryDest {
						return m.Source
					}
				}
			}
		}
	}
	return ""
}

func GetMountUpper(target string) (string, error) {
	cmd := fmt.Sprintf("mount | grep %s", target)
	result, err := utils.RunSimpleCmd(cmd)
	if err != nil {
		return "", err
	}
	data := strings.Split(result, ",upperdir=")
	if len(data) < 2 {
		return "", err
	}

	data = strings.Split(data[1], ",workdir=")
	return strings.TrimSpace(data[0]), nil
}

func IsMounted(target string) bool {
	cmd := fmt.Sprintf("mount | grep %s", target)
	result, err := utils.RunSimpleCmd(cmd)
	if err != nil {
		return false
	}
	if strings.Contains(result, target) {
		return true
	}
	return false
}

func IsAllPodsRunning() bool {
	// wait resource to sync
	time.Sleep(10 * time.Second)
	err := infraUtils.Retry(10, 5*time.Second, func() error {
		c, err := client.NewClientSet()
		if err != nil {
			return fmt.Errorf("failed to create k8s client  %v", err)
		}
		namespacePodList, err := client.ListAllNamespacesPods(c)
		if err != nil {
			return err
		}
		var notRunning int
		for _, podNamespace := range namespacePodList {
			for _, pod := range podNamespace.PodList.Items {
				if pod.Status.Phase != "Running" {
					logger.Info(podNamespace.Namespace.Name, pod.Name, pod.Status.Phase)
					notRunning++
					continue
				}
			}
		}
		if notRunning > 0 {
			logger.Info("remaining %d pod not running", notRunning)
			return fmt.Errorf("pod not running")
		}
		return nil
	})
	return err == nil
}
