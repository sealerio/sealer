// Copyright © 2022 Alibaba Group Holding Ltd.
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

package hostpool

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Output runs cmd on the remote host and returns its standard output.
// It must be executed in deploying node and towards the host instance.
func (host *Host) Output(cmd string) ([]byte, error) {
	if host.isLocal {
		return exec.Command(cmd).Output()
	}
	return host.sshSession.Output(cmd)
}

// CombinedOutput wraps the sshSession.CombinedOutput and does the same in both input and output.
// It must be executed in deploying node and towards the host instance.
func (host *Host) CombinedOutput(cmd string) ([]byte, error) {
	if host.isLocal {
		return exec.Command(cmd).CombinedOutput()
	}
	return host.sshSession.CombinedOutput(cmd)
}

// RunAndStderr runs a specified command and output stderr content.
// If command returns a nil, then no matter if there is content in session's stderr, just ignore stderr;
// If command return a non-nil, construct and return a new error with stderr content
// which may contains the exact error message.
//
// TODO: there is a potential issue that if much content is in stdout or stderr, and
// it may eventually cause the remote command to block.
//
// It must be executed in deploying node and towards the host instance.
func (host *Host) RunAndStderr(cmd string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	if host.isLocal {
		localCmd := exec.Command(cmd)
		localCmd.Stdout = &stdout
		localCmd.Stderr = &stderr
		if err := localCmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to exec cmd(%s) on host(%s): %s", cmd, host.config.IP, stderr.String())
		}
		return stdout.Bytes(), nil
	}

	host.sshSession.Stdout = &stdout
	host.sshSession.Stderr = &stderr
	if err := host.sshSession.Run(cmd); err != nil {
		return nil, fmt.Errorf("failed to exec cmd(%s) on host(%s): %s", cmd, host.config.IP, stderr.String())
	}

	return stdout.Bytes(), nil
}

// TODO: Do we need asynchronously output stdout and stderr?
