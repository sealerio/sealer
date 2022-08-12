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
	"fmt"
	"strings"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/utils/net"

	"github.com/sirupsen/logrus"
)

const SUDO = "sudo "

func (s *SSH) Ping(host string) error {
	if net.IsLocalIP(host, s.LocalAddress) {
		return nil
	}
	client, _, err := s.Connect(host)
	if err != nil {
		return fmt.Errorf("[ssh %s]create ssh session failed, %v", host, err)
	}
	err = client.Close()
	if err != nil {
		return err
	}
	return nil
}

func getKeyError(str string) string {
	begin := "Panic error: "
	end := ", please check this panic"
	n := strings.Index(str, begin)
	if n == -1 {
		return ""
	}

	m := strings.Index(str, end)
	if m == -1 {
		return ""
	}

	return str[n+len(begin) : m]
}

func (s *SSH) CmdAsync(host string, cmds ...string) ([]byte, error) {
	execFunc := func(cmd string) ([]byte, error) {
		client, session, err := s.Connect(host)
		if err != nil {
			return nil, fmt.Errorf("failed to create ssh session for %s: %v", host, err)
		}
		defer client.Close()
		defer session.Close()
		stdout, err := session.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout pipe for %s: %v", host, err)
		}
		stderr, err := session.StderrPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stderr pipe for %s: %v", host, err)
		}

		if err := session.Start(cmd); err != nil {
			return nil, fmt.Errorf("failed to start command %s on %s: %v", cmd, host, err)
		}

		output := make([]byte, 0)

		ReadPipe(stdout, stderr, &output)

		err = session.Wait()
		if err != nil {
			keyError := string(output)

			if ke := getKeyError(keyError); ke != "" {
				keyError = ke
			}
			return nil, fmt.Errorf("failed to execute command on host(%s): %v", host, keyError)
		}

		return output, nil
	}

	output := make([]byte, 0)
	for _, cmd := range cmds {
		if cmd == "" {
			continue
		}
		if s.User != common.ROOT {
			cmd = fmt.Sprintf("sudo -E /bin/sh <<EOF\n%s\nEOF", cmd)
		}
		o, err := execFunc(cmd)
		if err != nil {
			return nil, err
		}
		output = append(output, o...)
	}

	return output, nil
}

func (s *SSH) Cmd(host, cmd string) ([]byte, error) {
	if s.User != common.ROOT {
		cmd = fmt.Sprintf("sudo -E /bin/sh <<EOF\n%s\nEOF", cmd)
	}

	client, session, err := s.Connect(host)
	if err != nil {
		return nil, fmt.Errorf("[ssh][%s] create ssh session failed, %s", host, err)
	}
	defer client.Close()
	defer session.Close()
	b, err := session.CombinedOutput(cmd)
	if err != nil {
		keyError := string(b)
		logrus.Debugf("failed to execute command(%s) on host(%s): output(%s), error(%v)", cmd, host, keyError, err)

		if ke := getKeyError(keyError); ke != "" {
			keyError = ke
		}
		return nil, fmt.Errorf("failed to execute command on host(%s): %v", host, keyError)
	}

	return b, nil
}

//CmdToString is in host exec cmd and replace to spilt str
func (s *SSH) CmdToString(host, cmd, split string) (string, error) {
	data, err := s.Cmd(host, cmd)
	str := string(data)
	if err != nil {
		return str, err
	}
	if data != nil {
		str = strings.ReplaceAll(str, "\r\n", split)
		str = strings.ReplaceAll(str, "\n", split)
		return str, nil
	}
	return str, fmt.Errorf("command %s %s return nil", host, cmd)
}
