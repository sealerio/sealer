package build

import (
	"context"
	"fmt"
	"github.com/alibaba/sealer/runtime"
	"github.com/alibaba/sealer/utils/archive"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/alibaba/sealer/client"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image"
	infraUtils "github.com/alibaba/sealer/infra/utils"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
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

func setClusterFileToImage(image *v1.Image, name string) error {
	var cluster v1.Cluster
	clusterFileData := GetRawClusterFile(image)
	if clusterFileData == "" {
		return fmt.Errorf("failed to get cluster file from context or base image")
	}
	err := yaml.Unmarshal([]byte(clusterFileData), &cluster)
	if err != nil {
		return err
	}
	cluster.Spec.Image = name
	clusterFile, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}

	if image.Annotations == nil {
		image.Annotations = make(map[string]string)
	}

	image.Annotations[common.ImageAnnotationForClusterfile] = string(clusterFile)

	return nil
}

func IsOnlyCopy(layers []v1.Layer) bool {
	for i := 1; i < len(layers); i++ {
		if layers[i].Type == common.RUNCOMMAND ||
			layers[i].Type == common.CMDCOMMAND {
			return false
		}
	}
	return true
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

	opts := types.ContainerListOptions{All: true}
	opts.Filters = filters.NewArgs()
	opts.Filters.Add("name", registryName)
	containers, err := cli.ContainerList(ctx, opts)

	if err != nil {
		return ""
	}

	for _, c := range containers {
		for _, m := range c.Mounts {
			if m.Type == mount.TypeBind && m.Destination == registryDest {
				return m.Source
			}
		}
	}

	return ""
}

func GetMountDetails(target string) (mounted bool, upper string) {
	cmd := fmt.Sprintf("mount | grep %s", target)
	result, err := utils.RunSimpleCmd(cmd)
	if err != nil {
		return false, ""
	}
	if !strings.Contains(result, target) {
		return false, ""
	}

	data := strings.Split(result, ",upperdir=")
	if len(data) < 2 {
		return false, ""
	}

	data = strings.Split(data[1], ",workdir=")
	return true, strings.TrimSpace(data[0])
}

func IsAllPodsRunning() bool {
	err := infraUtils.Retry(10, 5*time.Second, func() error {
		c, err := client.NewClientSet()
		if err != nil {
			return fmt.Errorf("failed to create k8s client %v", err)
		}
		namespacePodList, err := client.ListAllNamespacesPods(c)
		if err != nil {
			return err
		}

		var notRunning int
		for _, podNamespace := range namespacePodList {
			logger.Info(podNamespace.Namespace.Name)
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

// parse context and kubefile. return context abs path and kubefile abs path
func ParseBuildArgs(localContextDir, kubeFileName string) (string, string, error) {
	localDir, err := resolveAndValidateContextPath(localContextDir)
	if err != nil {
		return "", "", err
	}

	if kubeFileName != "" {
		if kubeFileName, err = filepath.Abs(kubeFileName); err != nil {
			return "", "", fmt.Errorf("unable to get absolute path to KubeFile: %v", err)
		}
	}

	relFileName, err := getKubeFileRelPath(localDir, kubeFileName)
	return localDir, relFileName, err
}

func resolveAndValidateContextPath(givenContextDir string) (string, error) {
	absContextDir, err := filepath.Abs(givenContextDir)
	if err != nil {
		return "", fmt.Errorf("unable to get absolute context directory %s: %v", givenContextDir, err)
	}

	absContextDir, err = filepath.EvalSymlinks(absContextDir)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate symlinks in context path: %v", err)
	}

	stat, err := os.Lstat(absContextDir)
	if err != nil {
		return "", fmt.Errorf("unable to stat context directory %s: %v", absContextDir, err)
	}

	if !stat.IsDir() {
		return "", fmt.Errorf("context must be a directory: %s", absContextDir)
	}

	return absContextDir, err
}

func getKubeFileRelPath(absContextDir, givenKubeFile string) (string, error) {
	var err error

	absKubeFile := givenKubeFile
	if absKubeFile == "" {
		absKubeFile = filepath.Join(absContextDir, kubefile)
		if _, err = os.Lstat(absKubeFile); os.IsNotExist(err) {
			altPath := filepath.Join(absContextDir, strings.ToLower(kubefile))
			if _, err = os.Lstat(altPath); err == nil {
				absKubeFile = altPath
			}
		}
	}

	if !filepath.IsAbs(absKubeFile) {
		absKubeFile = filepath.Join(absContextDir, absKubeFile)
	}

	absKubeFile, err = filepath.EvalSymlinks(absKubeFile)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate symlinks in KubeFile path: %v", err)
	}

	if _, err := os.Lstat(absKubeFile); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("cannot locate KubeFile: %s", absKubeFile)
		}
		return "", fmt.Errorf("unable to stat KubeFile: %v", err)
	}

	return absKubeFile, nil
}

func ValidateContextDirectory(srcPath string) error {
	contextRoot, err := filepath.Abs(srcPath)
	if err != nil {
		return err
	}

	return filepath.Walk(contextRoot, func(filePath string, f os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("can't stat '%s'", filePath)
			}
			if os.IsNotExist(err) {
				return fmt.Errorf("file '%s' not found", filePath)
			}
			return err
		}

		if f.IsDir() {
			return nil
		}

		if f.Mode()&(os.ModeSymlink|os.ModeNamedPipe) != 0 {
			return nil
		}

		currentFile, err := os.Open(filePath)
		if err != nil && os.IsPermission(err) {
			return fmt.Errorf("no permission to read from '%s'", filePath)
		}
		currentFile.Close()

		return nil
	})
}

func tarBuildContext(kubeFilePath string, context string, tarFileName string) error {
	file, err := os.Create(tarFileName)
	if err != nil {
		return fmt.Errorf("failed to create %s, err: %v", tarFileName, err)
	}
	defer file.Close()

	var pathsToCompress []string
	pathsToCompress = append(pathsToCompress, kubeFilePath, context)
	tarReader, err := archive.TarWithoutRootDir(pathsToCompress...)
	if err != nil {
		return fmt.Errorf("failed to new tar reader when send build context, err: %v", err)
	}
	defer tarReader.Close()

	_, err = io.Copy(file, tarReader)
	if err != nil {
		return fmt.Errorf("failed to tar build context, err: %v", err)
	}
	return nil
}
