package testing

import (
	"fmt"
	"gitlab.alibaba-inc.com/seadent/pkg/build"
	"gitlab.alibaba-inc.com/seadent/pkg/utils/ssh"
	"os"
	"path/filepath"
	"testing"
)

/*func TestLocalBuilder_BuildOneByOne(t *testing.T) {
	lb := new(build.LocalBuilder)
	lb.KubefileName = "kubefile"
	lb.ImageName = "myimage"
	lb.Context = "."
	err := lb.InitImageSpec()
	if err != nil {
		t.Errorf("init images error %v\n", err)
	}

	t.Run("test build", func(t *testing.T) {
		err := lb.ExecBuildCopy()
		if err != nil {
			t.Errorf("exec build copy error %v\n", err)
		}

		cluster := &v1.Cluster{
			TypeMeta:   metav1.TypeMeta{APIVersion: "", Kind: "Image"},
			ObjectMeta: metav1.ObjectMeta{Name: "myCluster"},
		}

		lb.Cluster = cluster

		err = lb.ExecBuild()
		if err != nil {
			t.Errorf("exec build error %v\n", err)
		}

		want := []string{"dashboard.yaml", "prometheus.yaml", "helm", "redis.tar.gz", "aa"}

		var actual []string

		imagePath := filepath.Join(common.DefaultImageRootDir, lb.Image.Spec.ID)
		for i := range lb.Image.Spec.Layers {
			filename, err := FindFileInDir(filepath.Join(imagePath, lb.Image.Spec.Layers[i].Hash))
			if err != nil {
				t.Errorf("get actual file name failed %v\n", err)
			}
			actual = append(actual, filename[0])
		}

		if !reflect.DeepEqual(actual, want) {
			t.Errorf("copy file not Equal want = %v, actual = %v", want, actual)
		}
	})

}*/

func TestLocalBuilder_Build(t *testing.T) {

	conf := &build.Config{
		SSH: &ssh.SSH{
			User:     "a",
			Password: "b",
		},
	}
	builder := build.NewBuilder(conf, "")
	err := builder.Build("temp1", ".", "kubefile", "")
	if err != nil {
		t.Errorf("exec build error %v\n", err)
	}

}

func FindFileInDir(dirName string) ([]string, error) {
	var fileList []string
	err := filepath.Walk(dirName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("access path error %v", err)
		}
		if !info.IsDir() {
			fileList = append(fileList, info.Name())
		}
		return nil
	})

	return fileList, err
}
