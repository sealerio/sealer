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

package weed

import (
	"path"
	"strconv"

	"github.com/sealerio/sealer/utils/exec"
)

// checkPort checks if the port is available or can be used.
func checkPort(port int) bool {
	// lsof -i:9333
	err := exec.Cmd("lsof", "-i:"+strconv.Itoa(port))
	return err == nil
}

// checkDir checks if the dir is available or can be used.
//func checkDir(dir string) bool {
//	// ls /tmp
//	err := exec.Cmd("ls", dir)
//	if err != nil {
//		return false
//	}
//	return true
//}

func checkBinFile(fileName string) bool {
	binName := path.Base(fileName)
	switch binName {
	case "weed":

	case "etcd":

	default:
	}
	return false
}
