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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/pkg/define/application"

	"github.com/sealerio/sealer/pkg/define/options"

	parse2 "github.com/containers/buildah/pkg/parse"

	"github.com/sealerio/sealer/pkg/imageengine"

	v1 "github.com/sealerio/sealer/pkg/define/application/v1"

	"github.com/sealerio/sealer/pkg/define/application/version"

	"github.com/pkg/errors"

	"github.com/sealerio/sealer/build/kubefile/command"
)

// LegacyContext stores legacy information during the process of parsing.
// After the parsing ends, return the context to caller, and let the caller
// decide to clean.
type LegacyContext struct {
	files       []string
	directories []string
	// this is a map for appname to related-files
	// only used in test.
	apps2Files map[string][]string
}

type KubefileResult struct {
	Dockerfile    string
	LaunchList    []string
	Applications  map[string]version.VersionedApplication
	legacyContext LegacyContext
}

type KubefileParser struct {
	appRootPathFunc func(name string) string
	// path to build context
	buildContext string
	pullPolicy   string
	imageEngine  imageengine.Interface
}

func (kp *KubefileParser) ParseKubefile(rwc io.Reader) (*KubefileResult, error) {
	result, err := parse(rwc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dockerfile: %v", err)
	}

	mainNode := result.AST
	return kp.generateResult(mainNode)
}

func (kp *KubefileParser) generateResult(mainNode *Node) (*KubefileResult, error) {
	var (
		result = &KubefileResult{
			Applications: map[string]version.VersionedApplication{},
			legacyContext: LegacyContext{
				files:       []string{},
				directories: []string{},
				apps2Files:  map[string][]string{},
			},
			LaunchList: []string{},
		}

		err error

		launchCnt = 0
		cmdsCnt   = 0
	)

	defer func() {
		if err != nil {
			if err2 := result.CleanLegacyContext(); err2 != nil {
				logrus.Warn(err2)
			}
		}
	}()

	// pre-action for commands
	// for FROM, it will try to pull the image, and get apps from "FROM" image
	// for LAUNCH, it will check if it's the last line
	for i, node := range mainNode.Children {
		_command := node.Value
		if _, ok := command.SupportedCommands[_command]; !ok {
			return nil, errors.Errorf("command %s is not supported", _command)
		}

		switch _command {
		case command.From:
			// process FROM aims to pull the image, and merge the applications from
			// the FROM image.
			if err = kp.processFrom(node, result); err != nil {
				return nil, fmt.Errorf("failed to process from: %v", err)
			}
		case command.Launch:
			launchCnt++
			if launchCnt > 1 {
				return nil, errors.New("only one launch could be specified")
			}
			if i != len(mainNode.Children)-1 {
				return nil, errors.New("launch should be the last instruction")
			}

		case command.Cmds:
			cmdsCnt++
			if cmdsCnt > 1 {
				return nil, errors.New("only one cmds could be specified")
			}
			if i != len(mainNode.Children)-1 {
				return nil, errors.New("cmds should be the last instruction")
			}
		}

		if err = kp.processOnCmd(result, node); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (kp *KubefileParser) processOnCmd(result *KubefileResult, node *Node) error {
	cmd := node.Value
	switch cmd {
	case command.Label, command.Maintainer, command.Add, command.Arg, command.From, command.Copy, command.Run:
		result.Dockerfile = mergeLines(result.Dockerfile, node.Original)
		return nil
	case command.App:
		return kp.processApp(node, result)
	case command.Launch:
		return kp.processLaunch(node, result)
	case command.Cmds:
		return kp.processCmds(node, result)
	default:
		return fmt.Errorf("failed to recognize cmd: %s", cmd)
	}
}

func (kp *KubefileParser) processCmds(node *Node, result *KubefileResult) error {
	cmdsNode := node.Next
	for iter := cmdsNode; iter != nil; iter = iter.Next {
		result.LaunchList = append(result.LaunchList, iter.Value)
	}
	return nil
}

func (kp *KubefileParser) processLaunch(node *Node, result *KubefileResult) error {
	launchApp := func(appName string) (string, error) {
		appName = strings.TrimSpace(appName)
		app, ok := result.Applications[appName]
		if !ok {
			return "", errors.Errorf("application %s does not exist in the image", appName)
		}

		v1app := app.(*v1.Application)
		path := kp.appRootPathFunc(appName)
		switch v1app.Type() {
		case application.KubeApp:
			return fmt.Sprintf("kubectl apply -f %s", path), nil
		case application.HelmApp:
			return fmt.Sprintf("helm install %s %s", v1app.Name(), path), nil
		default:
			return "", errors.Errorf("unexpected application type %s", v1app.Type())
		}
	}

	appNode := node.Next
	for iter := appNode; iter != nil; iter = iter.Next {
		app := iter.Value
		lstr, err := launchApp(app)
		if err != nil {
			return err
		}
		result.LaunchList = append(result.LaunchList, lstr)
	}

	return nil
}

func (kp *KubefileParser) processFrom(node *Node, result *KubefileResult) error {
	var (
		platform  = parse2.DefaultPlatform()
		flags     = node.Flags
		imageNode = node.Next
	)
	if len(flags) > 0 {
		f, err := parseListFlag(flags[0])
		if err != nil {
			return err
		}
		if f.flag != "platform" {
			return errors.Errorf("flag %s is not available in FROM", f.flag)
		}
		platform = f.items[0]
	}

	if imageNode == nil || len(imageNode.Value) == 0 {
		return errors.Errorf("image should be specified in the FROM")
	}
	image := imageNode.Value
	if image == "scratch" {
		return nil
	}

	if err := kp.imageEngine.Pull(&options.PullOptions{
		PullPolicy: kp.pullPolicy,
		Image:      image,
		Platform:   platform,
	}); err != nil {
		return fmt.Errorf("failed to pull image %s: %v", image, err)
	}

	extension, err := kp.imageEngine.GetSealerImageExtension(&options.GetImageAnnoOptions{ImageNameOrID: image})
	if err != nil {
		return fmt.Errorf("failed to get image-extension %s: %s", image, err)
	}

	for _, app := range extension.Applications {
		// for range has problem.
		// can't assign address to the target.
		// we should use temp value.
		// https://github.com/golang/gofrontend/blob/e387439bfd24d5e142874b8e68e7039f74c744d7/go/statements.cc#L5501
		theApp := app
		result.Applications[app.Name()] = theApp
	}

	return nil
}

func (kr *KubefileResult) CleanLegacyContext() error {
	var (
		lc  = kr.legacyContext
		err error
	)

	for _, f := range lc.files {
		err = os.Remove(f)
	}

	for _, dir := range lc.directories {
		err = os.RemoveAll(dir)
	}

	return errors.Wrap(err, "failed to clean legacy context")
}

func NewParser(appRootPath string,
	buildOptions options.BuildOptions,
	imageEngine imageengine.Interface) *KubefileParser {
	return &KubefileParser{
		// application will be put under approot/name/
		appRootPathFunc: func(name string) string {
			return makeItDir(filepath.Join(appRootPath, name))
		},
		imageEngine:  imageEngine,
		buildContext: buildOptions.ContextDir,
		pullPolicy:   buildOptions.PullPolicy,
	}
}
