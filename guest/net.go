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

package guest

import (
	"bytes"
	"fmt"
	"text/template"
)

const (
	CALICO           = "calico"
	defaultCIDR      = "192.168.0.0/16"
	defaultInterface = "interface=eth.*|en.*"
)

type MetaData struct {
	//set IP auto-detection method
	Interface string
	CIDR      string
	//ipip mode for calico.yml,cannot be set at the same time as `vxlan`
	IPIP bool
	/*vxlan mode for calico.yml,cannot be set at the same time as `ipip`
	VXLAN bool*/
	//MTU size
	MTU string
}

type Net interface {
	// if template is "" using default template
	Manifests(template string) (string, error)
	// return cni template file
	Template() string
}

func render(data MetaData, templ string) (string, error) {
	var b bytes.Buffer
	t := template.Must(template.New("net").Parse(templ))
	if err := t.Execute(&b, &data); err != nil {
		return "", fmt.Errorf("render execute failed, %s", err)
	}
	return b.String(), nil
}

func NewNetWork(netName string, metaData MetaData) Net {
	switch netName {
	case CALICO:
		return &Calico{metaData}
	default:
		return &Calico{metaData}
	}
}
