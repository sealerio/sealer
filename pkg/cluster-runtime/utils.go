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

package clusterruntime

import (
	"fmt"
	"github.com/sealerio/sealer/utils"
	"net"
)

func confirmDeleteHosts(role string, nodesToDelete []net.IP) error {
	if !ForceDelete {
		if pass, err := utils.ConfirmOperation(fmt.Sprintf("Are you sure to delete these %s: %v? ", role, nodesToDelete)); err != nil {
			return err
		} else if !pass {
			return fmt.Errorf("exit the operation of delete these nodes")
		}
	}

	return nil
}
