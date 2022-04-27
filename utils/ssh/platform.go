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

package ssh

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/alibaba/sealer/utils"

	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/platform"
	"github.com/pkg/errors"
)

func (s *SSH) Platform(host string) (v1.Platform, error) {
	if utils.IsLocalIP(host, s.LocalAddress) {
		return *platform.GetDefaultPlatform(), nil
	}

	p := v1.Platform{}
	archResult, err := s.CmdToString(host, "uname -m", "")
	if err != nil {
		return p, err
	}
	osResult, err := s.CmdToString(host, "uname", "")
	if err != nil {
		return p, err
	}
	p.OS = strings.ToLower(strings.TrimSpace(osResult))
	switch strings.ToLower(strings.TrimSpace(archResult)) {
	case "x86_64":
		p.Architecture = platform.AMD
	case "aarch64":
		p.Architecture = platform.ARM64
	case "armv7l":
		p.Architecture = "arm-v7"
	case "armv6l":
		p.Architecture = "arm-v6"
	default:
		return p, fmt.Errorf("unrecognized architecture：%s", archResult)
	}
	if p.Architecture != platform.AMD {
		p.Variant, err = s.getCPUVariant(p.OS, p.Architecture, host)
		if err != nil {
			return p, err
		}
	}
	remotePlatform, err := platform.Parse(platform.Format(p))
	if err != nil {
		return p, err
	}
	return platform.Normalize(remotePlatform), nil
}

func (s *SSH) getCPUInfo(host, pattern string) (info string, err error) {
	sshClient, sftpClient, err := s.sftpConnect(host)
	if err != nil {
		return "", fmt.Errorf("new sftp client failed %v", err)
	}
	defer func() {
		_ = sftpClient.Close()
		_ = sshClient.Close()
	}()
	// open remote source file
	srcFile, err := sftpClient.Open("/proc/cpuinfo")
	if err != nil {
		return "", fmt.Errorf("open /proc/cpuinfo file failed %v", err)
	}
	defer func() {
		if err := srcFile.Close(); err != nil {
			logger.Warn("failed to close file: %v", err)
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
	return "", errors.Wrapf(platform.ErrNotFound, "getCPUInfo for pattern: %s", pattern)
}

func (s *SSH) getCPUVariant(os, arch, host string) (string, error) {
	variant, err := s.getCPUInfo(host, "Cpu architecture")
	if err != nil {
		return "", err
	}
	model, err := s.getCPUInfo(host, "model name")
	if err != nil {
		if !strings.Contains(err.Error(), platform.ErrNotFound.Error()) {
			return "", err
		}
	}
	variant, model = platform.NormalizeArch(variant, model)
	return platform.GetCPUVariantByInfo(os, arch, variant, model), nil
}
