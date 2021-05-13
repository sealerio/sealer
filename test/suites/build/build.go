package build

import (
	"path/filepath"

	"github.com/alibaba/sealer/test/testhelper"
)

func getFixtures() string {
	pwd := testhelper.GetPwd()
	return filepath.Join(pwd, "suites", "build", "fixtures")
}

func GetOnlyCopyDir() string {
	fixtures := getFixtures()
	return filepath.Join(fixtures, "only_copy")
}

func GetBuildTestDir() string {
	fixtures := getFixtures()
	return filepath.Join(fixtures, "build_test")
}
