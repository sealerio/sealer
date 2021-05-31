package image

import (
	"fmt"

	"github.com/alibaba/sealer/test/testhelper/settings"

	"github.com/alibaba/sealer/test/testhelper"
)

func DoImageOps(action, imageName string) {
	cmd := ""
	switch action {
	case "pull":
		cmd = fmt.Sprintf("%s pull %s", settings.DefaultSealerBin, imageName)
	case "push":
		cmd = fmt.Sprintf("%s push %s", settings.DefaultSealerBin, imageName)
	case "rmi":
		cmd = fmt.Sprintf("%s rmi %s", settings.DefaultSealerBin, imageName)
	case "run":
		cmd = fmt.Sprintf("%s run %s", settings.DefaultSealerBin, imageName)
	}

	testhelper.RunCmdAndCheckResult(cmd, 0)
}
func TagImages(oldName, newName string) {
	cmd := fmt.Sprintf("%s tag %s %s", settings.DefaultSealerBin, oldName, newName)
	testhelper.RunCmdAndCheckResult(cmd, 0)
}
