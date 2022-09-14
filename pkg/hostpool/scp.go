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
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CopyFile copies the contents of localFilePath to remote destination path.
// Both localFilePath and remotePath must be an absolute path.
//
// It must be executed in deploying node and towards the host instance.
func (host *Host) CopyToRemote(localFilePath string, remotePath string, permissions string) error {
	if host.isLocal {
		// TODO: add local file copy.
		return fmt.Errorf("local file copy is not implemented")
	}

	f, err := os.Open(filepath.Clean(localFilePath))
	if err != nil {
		return err
	}
	return host.scpClient.CopyFromFile(context.Background(), *f, remotePath, permissions)
}

// CopyFile copies the contents of remotePath to local destination path.
// Both localFilePath and remotePath must be an absolute path.
//
// It must be executed in deploying node and towards the host instance.
func (host *Host) CopyFromRemote(localFilePath string, remotePath string) error {
	if host.isLocal {
		// TODO: add local file copy.
		return fmt.Errorf("local file copy is not implemented")
	}

	f, err := os.Open(filepath.Clean(localFilePath))
	if err != nil {
		return err
	}
	return host.scpClient.CopyFromRemote(context.Background(), f, remotePath)
}

// CopyToRemoteDir copies the contents of local directory to remote destination directory.
// Both localFilePath and remotePath must be an absolute path.
//
// It must be executed in deploying node and towards the host instance.
func (host *Host) CopyToRemoteDir(localDir string, remoteDir string) error {
	if host.isLocal {
		// TODO: add local file copy.
		return fmt.Errorf("local file copy is not implemented")
	}

	// get the localDir Directory name
	fInfo, err := os.Lstat(localDir)
	if err != nil {
		return err
	}
	if !fInfo.IsDir() {
		return fmt.Errorf("input localDir(%s) is not a directory when copying directory content", localDir)
	}
	dirName := fInfo.Name()

	err = filepath.Walk(localDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Since localDir is an absolute path, then every passed path has a prefix of localDir,
		// then the relative path is the input path trims localDir.
		fileRelativePath := strings.TrimPrefix(path, localDir)
		remotePath := filepath.Join(remoteDir, dirName, fileRelativePath)

		return host.CopyToRemote(path, remotePath, info.Mode().String())
	})

	return err
}
