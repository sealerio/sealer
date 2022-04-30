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

package utils

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/image/store"
)

func SimilarImageListByName(imgName string) ([]string, error) {
	return similarImageList(imgName, true)
}

func similarImageList(imageArg string, byName bool) (similarImageList []string, err error) {
	is, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}
	metadataMap, err := is.GetImageMetadataMap()
	if err != nil {
		return nil, err
	}

	for name := range metadataMap {
		if byName && (strings.Contains(name, imageArg) || imageArg == "") {
			similarImageList = append(similarImageList, name)
		}
	}
	return
}

func ImageListFuncForCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	similarImages, err := SimilarImageListByName(toComplete)
	if err != nil {
		return nil, cobra.ShellCompDirectiveDefault
	}
	return similarImages, cobra.ShellCompDirectiveNoFileComp
}
