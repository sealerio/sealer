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
	"strings"

	"github.com/pkg/errors"
)

type listFlag struct {
	flag  string
	items []string
}

// parse --key=[a,b,c] or --key=a,b,c or --key="[a, b, c]"
// to listFlag {flag: key, items:[a, b, c]}
func parseListFlag(str string) (listFlag, error) {
	strs := strings.SplitN(strings.TrimLeft(str, "-"), "=", 2)
	if len(strs) < 2 {
		return listFlag{}, errors.New("flags should be like --flag=[value] or --flag=value")
	}
	key, values := strs[0], strs[1]
	values = strings.TrimLeft(values, "\"[")
	values = strings.TrimRight(values, "\"]")
	items := strings.Split(values, ",")
	if len(items) == 0 {
		return listFlag{}, errors.Errorf("empty input for flag %s is illegal", key)
	}
	for i, item := range items {
		items[i] = strings.TrimSpace(item)
	}

	return listFlag{
		flag:  key,
		items: items,
	}, nil
}
