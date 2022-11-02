// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exec

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/sealerio/sealer/common"

	"github.com/sirupsen/logrus"
)

const SUDO = "sudo"

func Cmd(name string, args ...string) error {
	username, err := GetCurrentUserName()
	if err != nil {
		return err
	}
	if username != common.ROOT {
		args = append([]string{name}, args...)
		name = SUDO
	}
	cmd := exec.Command(name, args[:]...) // #nosec
	cmd.Stdin = os.Stdin
	cmd.Stderr = common.StdErr
	cmd.Stdout = common.StdOut
	return cmd.Run()
}

func CmdOutput(name string, args ...string) ([]byte, error) {
	username, err := GetCurrentUserName()
	if err != nil {
		return nil, err
	}
	if username != common.ROOT {
		args = append([]string{name}, args...)
		name = SUDO
	}
	cmd := exec.Command(name, args[:]...) // #nosec
	return cmd.CombinedOutput()
}

func RunSimpleCmd(cmd string) (string, error) {
	username, err := GetCurrentUserName()
	if err != nil {
		return "", err
	}
	var result []byte
	if username != common.ROOT {
		result, err = exec.Command(SUDO, "/bin/bash", "-c", cmd).CombinedOutput() // #nosec
	} else {
		result, err = exec.Command("/bin/bash", "-c", cmd).CombinedOutput() // #nosec
	}
	if err != nil {
		logrus.Debugf("failed to execute command(%s): error(%v)", cmd, err)
	}
	return string(result), err
}

func CheckCmdIsExist(cmd string) (string, bool) {
	cmd = fmt.Sprintf("type %s", cmd)
	out, err := RunSimpleCmd(cmd)
	if err != nil {
		return "", false
	}

	outSlice := strings.Split(out, "is")
	last := outSlice[len(outSlice)-1]

	if last != "" && !strings.Contains(last, "not found") {
		return strings.TrimSpace(last), true
	}
	return "", false
}

func GetCurrentUserName() (string, error) {
	u, err := user.Current()
	return u.Username, err
}
