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

package platform

import (
	"bufio"
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
	v1 "github.com/sealerio/sealer/types/api/v1"
	utilsnet "github.com/sealerio/sealer/utils/net"
	"github.com/sealerio/sealer/utils/ssh"
	"github.com/sirupsen/logrus"
)

type RemotePlatform struct {
	sshRemote ssh.SSH
}

func (t *RemotePlatform) Platform(host net.IP) (v1.Platform, error) {
	if utilsnet.IsLocalIP(host, t.sshRemote.LocalAddress) {
		return *GetDefaultPlatform(), nil
	}

	p := v1.Platform{}
	archResult, err := t.sshRemote.CmdToString(host, "uname -m", "")
	if err != nil {
		return p, err
	}
	osResult, err := t.sshRemote.CmdToString(host, "uname", "")
	if err != nil {
		return p, err
	}
	p.OS = strings.ToLower(strings.TrimSpace(osResult))
	switch strings.ToLower(strings.TrimSpace(archResult)) {
	case "x86_64":
		p.Architecture = AMD
	case "aarch64":
		p.Architecture = ARM64
	case "armv7l":
		p.Architecture = "arm-v7"
	case "armv6l":
		p.Architecture = "arm-v6"
	default:
		return p, fmt.Errorf("unrecognized architecture: %s", archResult)
	}
	if p.Architecture != AMD {
		p.Variant, err = t.getCPUVariant(p.OS, p.Architecture, host)
		if err != nil {
			return p, err
		}
	}
	remotePlatform, err := Parse(Format(p))
	if err != nil {
		return p, err
	}
	return Normalize(remotePlatform), nil
}

func (t *RemotePlatform) getCPUInfo(host net.IP, pattern string) (info string, err error) {
	sshClient, sftpClient, err := t.sshRemote.SftpConnect(host)
	if err != nil {
		return "", fmt.Errorf("failed to new sftp client: %v", err)
	}
	defer func() {
		_ = sftpClient.Close()
		_ = sshClient.Close()
	}()
	// open remote source file
	srcFile, err := sftpClient.Open("/proc/cpuinfo")
	if err != nil {
		return "", fmt.Errorf("failed to open /proc/cpuinfo: %v", err)
	}
	defer func() {
		if err := srcFile.Close(); err != nil {
			logrus.Warnf("failed to close file: %v", err)
		}
	}()
	scanner := bufio.NewScanner(srcFile)
	for scanner.Scan() {
		newline := scanner.Text()
		list := strings.Split(newline, ":")

		if len(list) > 1 && strings.EqualFold(strings.TrimSpace(list[0]), pattern) {
			return strings.TrimSpace(list[1]), nil
		}
	}
	// Check whether the scanner encountered errors
	err = scanner.Err()
	if err != nil {
		return "", err
	}
	return "", errors.Wrapf(ErrNotFound, "getCPUInfo for pattern: %s", pattern)
}

func (t *RemotePlatform) getCPUVariant(os, arch string, host net.IP) (string, error) {
	variant, err := t.getCPUInfo(host, "Cpu architecture")
	if err != nil {
		return "", err
	}
	model, err := t.getCPUInfo(host, "model name")
	if err != nil {
		if !strings.Contains(err.Error(), ErrNotFound.Error()) {
			return "", err
		}
	}
	variant, model = NormalizeArch(variant, model)
	return GetCPUVariantByInfo(os, arch, variant, model), nil
}
