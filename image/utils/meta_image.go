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

	"github.com/alibaba/sealer/image/store"

	"github.com/spf13/cobra"
)

func SimilarImageList(imageArg string) (similarImageList []string, err error) {
	is, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}

	metadataMap, err := is.GetImageMetadataMap()
	if err != nil {
		return nil, err
	}
	for _, imageMetadata := range metadataMap {
		imageMeta := imageMetadata
		if !strings.Contains(imageMeta.Name, imageArg) && imageArg != "" {
			continue
		}
		similarImageList = append(similarImageList, imageMeta.Name)
	}
	return similarImageList, nil
}

func ImageListFuncForCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	similarImages, err := SimilarImageList(toComplete)
	if err != nil {
		return nil, cobra.ShellCompDirectiveDefault
	}
	return similarImages, cobra.ShellCompDirectiveNoFileComp
}
