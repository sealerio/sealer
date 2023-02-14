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

package buildah

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/containers/buildah/pkg/formats"
	"github.com/containers/common/libimage"
	"github.com/docker/go-units"
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/pkg/define/options"
)

const none = "<none>"

type jsonImage struct {
	ID           string    `json:"id"`
	Names        []string  `json:"names"`
	Digest       string    `json:"digest"`
	CreatedAt    string    `json:"createdat"`
	Size         string    `json:"size"`
	Created      int64     `json:"created"`
	CreatedAtRaw time.Time `json:"createdatraw"`
	ReadOnly     bool      `json:"readonly"`
	History      []string  `json:"history"`
}

type imageOutputParams struct {
	Tag          string
	ID           string
	Name         string
	Digest       string
	Created      int64
	CreatedAt    string
	Size         string
	CreatedAtRaw time.Time
	ReadOnly     bool
	History      string
}

type imageOptions struct {
	all       bool
	digests   bool
	format    string
	json      bool
	noHeading bool
	truncate  bool
	quiet     bool
	readOnly  bool
	history   bool
}

var imagesHeader = map[string]string{
	"Name":      "REPOSITORY",
	"Tag":       "TAG",
	"ID":        "IMAGE ID",
	"CreatedAt": "CREATED",
	"Size":      "SIZE",
	"ReadOnly":  "R/O",
	"History":   "HISTORY",
}

func (engine *Engine) Images(opts *options.ImagesOptions) error {
	runtime := engine.ImageRuntime()
	options := &libimage.ListImagesOptions{}
	if !opts.All {
		options.Filters = append(options.Filters, "intermediate=false")
		//options.Filters = append(options.Filters, "label=io.sealer.version")
	}

	//TODO add some label to identify sealer image and oci image.
	images, err := runtime.ListImages(getContext(), []string{}, options)
	if err != nil {
		return err
	}

	imageOpts := imageOptions{
		all:       opts.All,
		digests:   opts.Digests,
		json:      opts.JSON,
		noHeading: opts.NoHeading,
		truncate:  !opts.NoTrunc,
		quiet:     opts.Quiet,
		history:   opts.History,
	}

	if opts.JSON {
		return formatImagesJSON(images, imageOpts)
	}

	return formatImages(images, imageOpts)
}

func outputHeader(opts imageOptions) string {
	if opts.format != "" {
		return strings.Replace(opts.format, `\t`, "\t", -1)
	}
	if opts.quiet {
		return formats.IDString
	}
	format := "table {{.Name}}\t{{.Tag}}\t"
	if opts.noHeading {
		format = "{{.Name}}\t{{.Tag}}\t"
	}

	if opts.digests {
		format += "{{.Digest}}\t"
	}
	format += "{{.ID}}\t{{.CreatedAt}}\t{{.Size}}"
	if opts.readOnly {
		format += "\t{{.ReadOnly}}"
	}
	if opts.history {
		format += "\t{{.History}}"
	}
	return format
}

func formatImagesJSON(images []*libimage.Image, opts imageOptions) error {
	jsonImages := []jsonImage{}
	for _, image := range images {
		// Copy the base data over to the output param.
		size, err := image.Size()
		if err != nil {
			return err
		}
		created := image.Created()
		jsonImages = append(jsonImages,
			jsonImage{
				CreatedAtRaw: created,
				Created:      created.Unix(),
				CreatedAt:    units.HumanDuration(time.Since(created)) + " ago",
				Digest:       image.Digest().String(),
				ID:           TruncateID(image.ID(), opts.truncate),
				Names:        image.Names(),
				ReadOnly:     image.IsReadOnly(),
				Size:         formattedSize(size),
			})
	}

	data, err := json.MarshalIndent(jsonImages, "", "    ")
	if err != nil {
		return err
	}
	logrus.Infof("%s", data)
	return nil
}

type imagesSorted []imageOutputParams

func (a imagesSorted) Less(i, j int) bool {
	return a[i].CreatedAtRaw.After(a[j].CreatedAtRaw)
}

func (a imagesSorted) Len() int {
	return len(a)
}

func (a imagesSorted) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func formatImages(images []*libimage.Image, opts imageOptions) error {
	var outputData imagesSorted

	for _, image := range images {
		var outputParam imageOutputParams
		size, err := image.Size()
		if err != nil {
			return err
		}
		created := image.Created()
		outputParam.CreatedAtRaw = created
		outputParam.Created = created.Unix()
		outputParam.CreatedAt = units.HumanDuration(time.Since(created)) + " ago"
		outputParam.Digest = image.Digest().String()
		outputParam.ID = TruncateID(image.ID(), opts.truncate)
		outputParam.Size = formattedSize(size)
		outputParam.ReadOnly = image.IsReadOnly()

		repoTags, err := image.NamedRepoTags()
		if err != nil {
			return err
		}

		nameTagPairs, err := libimage.ToNameTagPairs(repoTags)
		if err != nil {
			return err
		}

		for _, pair := range nameTagPairs {
			newParam := outputParam
			newParam.Name = pair.Name
			newParam.Tag = pair.Tag
			newParam.History = formatHistory(image.NamesHistory(), pair.Name, pair.Tag)
			outputData = append(outputData, newParam)
			// `images -q` should a given ID only once.
			if opts.quiet {
				break
			}
		}
	}

	sort.Sort(outputData)
	out := formats.StdoutTemplateArray{Output: imagesToGeneric(outputData), Template: outputHeader(opts), Fields: imagesHeader}
	return formats.Writer(out).Out()
}

func formatHistory(history []string, name, tag string) string {
	if len(history) == 0 {
		return none
	}
	// Skip the first history entry if already existing as name
	if fmt.Sprintf("%s:%s", name, tag) == history[0] {
		if len(history) == 1 {
			return none
		}
		return strings.Join(history[1:], ", ")
	}
	return strings.Join(history, ", ")
}

func TruncateID(id string, truncate bool) string {
	if !truncate {
		return "sha256:" + id
	}

	if idTruncLength := 12; len(id) > idTruncLength {
		return id[:idTruncLength]
	}
	return id
}

func imagesToGeneric(templParams []imageOutputParams) (genericParams []interface{}) {
	if len(templParams) > 0 {
		for _, v := range templParams {
			genericParams = append(genericParams, interface{}(v))
		}
	}
	return genericParams
}

func formattedSize(size int64) string {
	suffixes := [5]string{"B", "KB", "MB", "GB", "TB"}

	count := 0
	formattedSize := float64(size)
	for formattedSize >= 1000 && count < 4 {
		formattedSize /= 1000
		count++
	}
	return fmt.Sprintf("%.3g %s", formattedSize, suffixes[count])
}

//func matchesID(imageID, argID string) bool {
//	return strings.HasPrefix(imageID, argID)
//}
//
//func matchesReference(name, argName string) bool {
//	if argName == "" {
//		return true
//	}
//	splitName := strings.Split(name, ":")
//	// If the arg contains a tag, we handle it differently than if it does not
//	if strings.Contains(argName, ":") {
//		splitArg := strings.Split(argName, ":")
//		return strings.HasSuffix(splitName[0], splitArg[0]) && (splitName[1] == splitArg[1])
//	}
//	return strings.HasSuffix(splitName[0], argName)
//}
