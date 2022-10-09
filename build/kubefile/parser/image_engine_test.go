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
	v12 "github.com/sealerio/sealer/pkg/define/application/v1"
	"github.com/sealerio/sealer/pkg/define/application/version"
	v1 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/define/options"
)

type testImageEngine struct{}

var testExtensionWithApp = v1.ImageExtension{
	Applications: []version.VersionedApplication{
		v12.NewV1Application(
			"es",
			"helm",
		),
		v12.NewV1Application(
			"ts",
			"kube",
		),
	},
}

var testExtensionWithoutApp = v1.ImageExtension{}

var testImage map[string]v1.ImageExtension = map[string]v1.ImageExtension{
	"withApp":    testExtensionWithApp,
	"withoutApp": testExtensionWithoutApp,
}

func (testImageEngine) Build(sealerBuildFlags *options.BuildOptions) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) CreateContainer(opts *options.FromOptions) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) Mount(opts *options.MountOptions) ([]options.JSONMount, error) {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) Copy(opts *options.CopyOptions) error {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) Commit(opts *options.CommitOptions) error {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) Login(opts *options.LoginOptions) error {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) Logout(opts *options.LogoutOptions) error {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) Push(opts *options.PushOptions) error {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) Pull(opts *options.PullOptions) error {
	return nil
}

func (testImageEngine) Images(opts *options.ImagesOptions) error {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) Save(opts *options.SaveOptions) error {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) Load(opts *options.LoadOptions) error {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) Inspect(opts *options.InspectOptions) error {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) GetImageAnnotation(opts *options.GetImageAnnoOptions) (map[string]string, error) {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) RemoveImage(opts *options.RemoveImageOptions) error {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) RemoveContainer(opts *options.RemoveContainerOptions) error {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) Tag(opts *options.TagOptions) error {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) CreateWorkingContainer(opts *options.BuildRootfsOptions) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (testImageEngine) GetSealerImageExtension(opts *options.GetImageAnnoOptions) (v1.ImageExtension, error) {
	_, ok := testImage[opts.ImageNameOrID]
	if !ok {
		return testExtensionWithoutApp, nil
	}
	return testExtensionWithApp, nil
}
