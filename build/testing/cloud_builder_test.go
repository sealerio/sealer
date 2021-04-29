package testing

import (
	"gitlab.alibaba-inc.com/seadent/pkg/build"
	"gitlab.alibaba-inc.com/seadent/pkg/utils/ssh"
	"os"
	"testing"
)

/*func TestCloudBuilder_BuildOneByOne(t *testing.T) {
	cb := new(build.CloudBuilder)
	cb.KubefileName = "kubefile"
	cb.ImageName = "myimage"
	cb.Context = "."
	err := cb.InitImageSpec()
	if err != nil {
		t.Errorf("init images error %v\n", err)
	}

	cluster := &v1.Cluster{
		TypeMeta:   metav1.TypeMeta{APIVersion: "", Kind: "Image"},
		ObjectMeta: metav1.ObjectMeta{Name: "myCluster"},
		Spec: v1.ClusterSpec{
			Masters: v1.Hosts{
				IPList: []string{"192.168.56.101", "192.168.56.102"},
			},
		},
	}

	cb.Cluster = cluster

	cb.SSH = &ssh.SSH{
		User:     "root",
		Password: "123456",
	}

	t.Run("test send context file", func(t *testing.T) {
		if err := cb.SendBuildContext(); err != nil {
			t.Errorf("send context failed,error is %v\n", err)
		}
		workdir := fmt.Sprintf(common.DefaultWorkDir, cb.Cluster.Name)
		want := filepath.Join(workdir, "kubefile")

		if !cb.SSH.IsFileExist(cb.Cluster.Annotations[common.RemoteServerEIPAnnotation], want) {
			t.Errorf("test send context file failed: %s not found", want)
		}
	})

	t.Run("test exec remote local build", func(t *testing.T) {
		if err := cb.RemoteLocalBuild(); err != nil {
			t.Errorf("remote local build failed,error is %v\n", err)
		}
	})

	t.Run("test PullImage from remote", func(t *testing.T) {
		if err := cb.SaveAndPullImage(); err != nil {
			t.Errorf("pull image failed,error is %v\n", err)
		}
		imageFileName := fmt.Sprintf("%s.tar.gz", cb.Image.Spec.ID)
		want := fmt.Sprintf("%s/%s", common.DefaultImageRootDir, imageFileName)
		if IsNotExist(want) {
			t.Errorf("test pull images file failed: %s not found", want)
		}
	})

}*/

func TestCloudBuilder_Build(t *testing.T) {
	conf := &build.Config{
		SSH: &ssh.SSH{
			User:     "a",
			Password: "b",
		},
	}

	/*	cluster := &v1.Cluster{
		TypeMeta:   metav1.TypeMeta{APIVersion: "", Kind: "Image"},
		ObjectMeta: metav1.ObjectMeta{Name: "myCluster"},
		Spec: v1.ClusterSpec{
			Masters: v1.Hosts{
				IPList: []string{"192.168.56.101", "192.168.56.102"},
			},
		},
	}*/

	builder := build.NewBuilder(conf, "cloud")

	err := builder.Build("myimage", ".", "kubefile", "88888888888")
	if err != nil {
		t.Errorf("exec build error %v\n", err)
	}

}

func IsNotExist(fileName string) bool {
	_, err := os.Lstat(fileName)
	return os.IsNotExist(err)
}
