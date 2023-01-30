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

package shellcommand

import (
	"fmt"
)

const DefaultSealerHostAliasAnnotation = "#hostalias-set-by-sealer"

func CommandSetHostAlias(hostName, ip string) string {
	return fmt.Sprintf(`if grep " %s " /etc/hosts &>/dev/null;then sed -i "/\ %s\ /d" /etc/hosts; fi;echo "%s %s %s" >>/etc/hosts`, hostName, hostName, ip, hostName, DefaultSealerHostAliasAnnotation)
}

func CommandUnSetHostAlias() string {
	return fmt.Sprintf(`echo "$(sed "/%s/d" /etc/hosts)" > /etc/hosts`, DefaultSealerHostAliasAnnotation)
}
