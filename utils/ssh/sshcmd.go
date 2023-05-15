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

package ssh

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/env"
	utilsnet "github.com/sealerio/sealer/utils/net"
)

const SUDO = "sudo "

func (s *SSH) Ping(host net.IP) error {
	if utilsnet.IsLocalIP(host, s.LocalAddress) {
		return nil
	}
	client, _, err := s.Connect(host)
	if err != nil {
		return fmt.Errorf("failed to ping node %s using ssh session: %v", host, err)
	}
	err = client.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *SSH) CmdAsync(host net.IP, hostEnv map[string]string, cmds ...string) error {
	// force specify PATH env
	if hostEnv == nil {
		hostEnv = map[string]string{}
	}
	if hostEnv["PATH"] == "" {
		hostEnv["PATH"] = "/sbin:/bin:/usr/sbin:/usr/bin:/usr/local/bin"
	}

	var execFunc func(cmd string) error

	if utilsnet.IsLocalIP(host, s.LocalAddress) {
		execFunc = func(cmd string) error {
			c := exec.Command("/bin/bash", "-c", cmd)
			stdout, err := c.StdoutPipe()
			if err != nil {
				return err
			}

			stderr, err := c.StderrPipe()
			if err != nil {
				return err
			}

			if err := c.Start(); err != nil {
				return fmt.Errorf("failed to start command %s: %v", cmd, err)
			}

			ReadPipe(stdout, stderr, s.AlsoToStdout)

			err = c.Wait()
			if err != nil {
				return fmt.Errorf("failed to execute command(%s) on host(%s): error(%v)", cmd, host, err)
			}
			return nil
		}
	} else {
		execFunc = func(cmd string) error {
			client, session, err := s.Connect(host)
			if err != nil {
				return fmt.Errorf("failed to create ssh session for %s: %v", host, err)
			}
			defer client.Close()
			defer session.Close()
			stdout, err := session.StdoutPipe()
			if err != nil {
				return fmt.Errorf("failed to create stdout pipe for %s: %v", host, err)
			}
			stderr, err := session.StderrPipe()
			if err != nil {
				return fmt.Errorf("failed to create stderr pipe for %s: %v", host, err)
			}

			if err := session.Start(cmd); err != nil {
				return fmt.Errorf("failed to start command %s on %s: %v", cmd, host, err)
			}

			ReadPipe(stdout, stderr, s.AlsoToStdout)

			err = session.Wait()
			if err != nil {
				return fmt.Errorf("failed to execute command(%s) on host(%s): error(%v)", cmd, host, err)
			}

			return nil
		}
	}

	for _, cmd := range cmds {
		if cmd == "" {
			continue
		}
		if s.User != common.ROOT {
			cmd = fmt.Sprintf("sudo -E /bin/bash <<EOF\n%s\nEOF", cmd)
		}
		cmd = env.WrapperShell(cmd, hostEnv)

		if err := execFunc(cmd); err != nil {
			return err
		}
	}

	return nil
}

func (s *SSH) Cmd(host net.IP, hostEnv map[string]string, cmd string) ([]byte, error) {
	// force specify PATH env
	if hostEnv == nil {
		hostEnv = map[string]string{}
	}
	if hostEnv["PATH"] == "" {
		hostEnv["PATH"] = "/sbin:/bin:/usr/sbin:/usr/bin:/usr/local/bin"
	}

	if s.User != common.ROOT {
		cmd = fmt.Sprintf("sudo -E /bin/bash <<EOF\n%s\nEOF", cmd)
	}
	cmd = env.WrapperShell(cmd, hostEnv)

	var stdoutContent, stderrContent bytes.Buffer

	if utilsnet.IsLocalIP(host, s.LocalAddress) {
		localCmd := exec.Command("/bin/bash", "-c", cmd)
		localCmd.Stdout = &stdoutContent
		localCmd.Stderr = &stderrContent
		if err := localCmd.Run(); err != nil {
			return stdoutContent.Bytes(), fmt.Errorf("failed to execute command(%s) on host(%s): error(%v)", cmd, host, stderrContent.String())
		}
		return stdoutContent.Bytes(), nil
	}

	client, session, err := s.Connect(host)
	if err != nil {
		return nil, fmt.Errorf("[ssh][%s] failed to create ssh session: %s", host, err)
	}
	defer client.Close()
	defer session.Close()

	session.Stdout = &stdoutContent
	session.Stderr = &stderrContent
	if err := session.Run(cmd); err != nil {
		return stdoutContent.Bytes(), fmt.Errorf("[ssh][%s]failed to run command[%s]: %s", host, cmd, stderrContent.String())
	}

	return stdoutContent.Bytes(), nil
}

// CmdToString is in host exec cmd and replace to spilt str
func (s *SSH) CmdToString(host net.IP, env map[string]string, cmd, split string) (string, error) {
	data, err := s.Cmd(host, env, cmd)
	str := string(data)
	if err != nil {
		return str, err
	}
	if data != nil {
		str = strings.ReplaceAll(str, "\r", split)
		str = strings.ReplaceAll(str, "\r\n", split)
		str = strings.ReplaceAll(str, "\n", split)
		return str, nil
	}
	return str, fmt.Errorf("command %s %s return nil", host, cmd)
}
