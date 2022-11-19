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

package parser

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/sealerio/sealer/build/kubefile/command"
	v1 "github.com/sealerio/sealer/pkg/define/application/v1"
	"github.com/sealerio/sealer/pkg/define/application/version"
)

func (kp *KubefileParser) processApp(node *Node, result *KubefileResult) (version.VersionedApplication, error) {
	var (
		appName     = ""
		localFiles  = []string{}
		remoteFiles = []string{}
		filesToCopy = []string{}
	)

	// first node value is the command
	for ptr := node.Next; ptr != nil; ptr = ptr.Next {
		val := ptr.Value
		// record the first word to be the app name
		if appName == "" {
			appName = val
			continue
		}
		switch {
		//
		case isLocal(val):
			localFiles = append(localFiles, trimLocal(val))
		case isRemote(val):
			remoteFiles = append(remoteFiles, val)
		default:
			return nil, errors.New("source schema should be specified with https:// http:// local:// in APP")
		}
	}

	if appName == "" {
		return nil, errors.New("app name should be specified in the app cmd")
	}

	// TODO clean the app directory first before putting files into it.
	// this will rely on the storage interface
	if len(localFiles) > 0 {
		filesToCopy = append(filesToCopy, localFiles...)
	}

	// for the remote files
	// 1. create a temp dir under the build context
	// 2. download remote files to the temp dir
	// 3. append the temp files to filesToCopy
	if len(remoteFiles) > 0 {
		tmpDir, err := os.MkdirTemp(kp.buildContext, "sealer-remote-files")
		if err != nil {
			return nil, errors.Errorf("failed to create remote context: %s", err)
		}

		files, err := downloadRemoteFiles(tmpDir, remoteFiles)
		if err != nil {
			return nil, err
		}

		filesToCopy = append(filesToCopy, files...)
		// append it to the legacy.
		// it will be deleted by CleanContext
		result.legacyContext.directories = append(result.legacyContext.directories, tmpDir)
	}

	destDir := kp.appRootPathFunc(appName)
	tmpLine := strings.Join(append([]string{command.Copy}, append(filesToCopy, destDir)...), " ")
	result.Dockerfile = mergeLines(result.Dockerfile, tmpLine)
	result.legacyContext.apps2Files[appName] = append([]string{}, filesToCopy...)

	return makeItAsApp(appName, filesToCopy, result)
}

func makeItAsApp(appName string, filesToJudge []string, result *KubefileResult) (version.VersionedApplication, error) {
	appType, err := getApplicationType(filesToJudge)
	if err != nil {
		return nil, fmt.Errorf("failed to judge the application type for %s: %v", appName, err)
	}

	launchFiles, err := getApplicationFiles(appName, appType, filesToJudge)
	if err != nil {
		return nil, fmt.Errorf("failed to get app (%s)launch files: %v", appName, err)
	}

	v1App := v1.NewV1Application(
		appName,
		appType,
		launchFiles,
	).(*v1.Application)
	result.Applications[v1App.Name()] = v1App
	return v1App, nil
}

func downloadRemoteFiles(shadowDir string, files []string) ([]string, error) {
	var (
		downloaded = []string{}
		err        error
	)

	for _, src := range files {
		var filePath string
		filePath, err = getFileFromURL(src, "", shadowDir)
		if err != nil {
			return nil, errors.Errorf("failed to download file %s, %s", src, err)
		}
		downloaded = append(downloaded, filePath)
	}
	return downloaded, nil
}
