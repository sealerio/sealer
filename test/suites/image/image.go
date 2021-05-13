package image

import (
	"fmt"

	"github.com/alibaba/sealer/test/testhelper"
)

func DoImageOps(action, imageName string) {
	cmd := ""
	switch action {
	case "pull":
		cmd = fmt.Sprintf("sealer pull %s", imageName)
	case "push":
		cmd = fmt.Sprintf("sealer push %s", imageName)
	case "rmi":
		cmd = fmt.Sprintf("sealer rmi %s", imageName)
	case "run":
		cmd = fmt.Sprintf("sealer run %s", imageName)
	}

	testhelper.RunCmdAndCheckResult(cmd, 0)
}
func TagImages(oldName, newName string) {
	cmd := fmt.Sprintf("sealer tag %s %s", oldName, newName)
	testhelper.RunCmdAndCheckResult(cmd, 0)
}
