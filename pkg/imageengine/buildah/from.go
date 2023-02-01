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
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/pkg/config"
	encconfig "github.com/containers/ocicrypt/config"

	"github.com/pkg/errors"

	"github.com/sealerio/sealer/pkg/define/options"
)

type fromFlagsWrapper struct {
	*buildahcli.FromAndBudResults
	*buildahcli.UserNSResults
	*buildahcli.NameSpaceResults
}

// createContainerFromImage create a working container. This function is copied from
// "buildah from". This function takes args([]string{"$image"}), and create a working container
// based on $image, this will generate an empty dictionary, not a real rootfs. And this container is a fake container.
func (engine *Engine) createContainerFromImage(opts *options.FromOptions) (string, error) {
	defaultContainerConfig, err := config.Default()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get container config")
	}

	if len(opts.Image) == 0 {
		return "", errors.Errorf("an image name (or \"scratch\") must be specified")
	}

	// TODO be aware of this, maybe this will incur platform problem.
	systemCxt := engine.SystemContext()

	// TODO we do not support from remote currently
	// which is to make the policy pull-if-missing
	pullPolicy := define.PullNever

	store := engine.ImageStore()

	commonOpts, err := parse.CommonBuildOptions(engine.Command)
	if err != nil {
		return "", err
	}

	isolation, err := defaultIsolationOption()
	if err != nil {
		return "", err
	}

	namespaceOptions, networkPolicy := defaultNamespaceOptions()

	usernsOption, idmappingOptions, err := parse.IDMappingOptions(engine.Command, isolation)
	if err != nil {
		return "", errors.Wrapf(err, "error parsing ID mapping options")
	}
	namespaceOptions.AddOrReplace(usernsOption...)

	// hardcode format here, user do not concern about this.
	format, err := getImageType(define.OCI)
	if err != nil {
		return "", err
	}

	capabilities, err := defaultContainerConfig.Capabilities("", []string{}, []string{})
	if err != nil {
		return "", err
	}

	commonOpts.Ulimit = append(defaultContainerConfig.Containers.DefaultUlimits, commonOpts.Ulimit...)

	options := buildah.BuilderOptions{
		FromImage:             opts.Image,
		Container:             "",
		ContainerSuffix:       "",
		PullPolicy:            pullPolicy,
		SystemContext:         systemCxt,
		DefaultMountsFilePath: "",
		Isolation:             isolation,
		NamespaceOptions:      namespaceOptions,
		ConfigureNetwork:      networkPolicy,
		CNIPluginPath:         "",
		CNIConfigDir:          "",
		IDMappingOptions:      idmappingOptions,
		Capabilities:          capabilities,
		CommonBuildOpts:       commonOpts,
		Format:                format,
		DefaultEnv:            defaultContainerConfig.GetDefaultEnv(),
		MaxPullRetries:        maxPullPushRetries,
		PullRetryDelay:        pullPushRetryDelay,
		OciDecryptConfig:      &encconfig.DecryptConfig{},
	}

	if !opts.Quiet {
		options.ReportWriter = os.Stderr
	}

	builder, err := buildah.NewBuilder(getContext(), store, options)
	if err != nil {
		return "", err
	}

	if err := onBuild(builder, opts.Quiet); err != nil {
		return "", err
	}

	return builder.ContainerID, builder.Save()
}

func (engine *Engine) CreateContainer(opts *options.FromOptions) (string, error) {
	wrapper := &fromFlagsWrapper{
		FromAndBudResults: &buildahcli.FromAndBudResults{},
		UserNSResults:     &buildahcli.UserNSResults{},
		NameSpaceResults:  &buildahcli.NameSpaceResults{},
	}

	flags := engine.Flags()
	fromAndBudFlags, err := buildahcli.GetFromAndBudFlags(wrapper.FromAndBudResults, wrapper.UserNSResults, wrapper.NameSpaceResults)
	if err != nil {
		return "", err
	}

	flags.AddFlagSet(&fromAndBudFlags)

	err = engine.migrateFlags2BuildahFrom(opts)
	if err != nil {
		return "", err
	}

	return engine.createContainerFromImage(opts)
}

// CreateWorkingContainer will make a workingContainer with rootfs under /var/lib/containers/storage
// And then link rootfs to the DestDir
// And remember to call RemoveContainer to remove the link and remove the container(umount rootfs) manually.
func (engine *Engine) CreateWorkingContainer(opts *options.BuildRootfsOptions) (containerID string, err error) {
	// TODO clean environment when it fails
	cid, err := engine.CreateContainer(&options.FromOptions{
		Image: opts.ImageNameOrID,
		Quiet: false,
	})
	if err != nil {
		return "", err
	}

	mounts, err := engine.Mount(&options.MountOptions{Containers: []string{cid}})
	if err != nil {
		return "", err
	}

	// remove destination dir if it exists, otherwise the Symlink will fail.
	if _, err = os.Stat(opts.DestDir); err == nil {
		return "", fmt.Errorf("destination directionay %s exists, you should remove it first", opts.DestDir)
	}

	mountPoint := mounts[0].MountPoint
	return cid, os.Symlink(mountPoint, opts.DestDir)
}

func (engine *Engine) migrateFlags2BuildahFrom(opts *options.FromOptions) error {
	return nil
}

func onBuild(builder *buildah.Builder, quiet bool) error {
	ctr := 0
	for _, onBuildSpec := range builder.OnBuild() {
		ctr = ctr + 1
		commands := strings.Split(onBuildSpec, " ")
		command := strings.ToUpper(commands[0])
		args := commands[1:]
		if !quiet {
			fmt.Fprintf(os.Stderr, "STEP %d: %s\n", ctr, onBuildSpec)
		}
		switch command {
		case "ADD":
		case "COPY":
			dest := ""
			srcs := []string{}
			size := len(args)
			if size > 1 {
				dest = args[size-1]
				srcs = args[:size-1]
			}
			if err := builder.Add(dest, command == "ADD", buildah.AddAndCopyOptions{}, srcs...); err != nil {
				return err
			}
		case "ANNOTATION":
			annotation := strings.SplitN(args[0], "=", 2)
			if len(annotation) > 1 {
				builder.SetAnnotation(annotation[0], annotation[1])
			} else {
				builder.UnsetAnnotation(annotation[0])
			}
		case "CMD":
			builder.SetCmd(args)
		case "ENV":
			env := strings.SplitN(args[0], "=", 2)
			if len(env) > 1 {
				builder.SetEnv(env[0], env[1])
			} else {
				builder.UnsetEnv(env[0])
			}
		case "ENTRYPOINT":
			builder.SetEntrypoint(args)
		case "EXPOSE":
			builder.SetPort(strings.Join(args, " "))
		case "HOSTNAME":
			builder.SetHostname(strings.Join(args, " "))
		case "LABEL":
			label := strings.SplitN(args[0], "=", 2)
			if len(label) > 1 {
				builder.SetLabel(label[0], label[1])
			} else {
				builder.UnsetLabel(label[0])
			}
		case "MAINTAINER":
			builder.SetMaintainer(strings.Join(args, " "))
		case "ONBUILD":
			builder.SetOnBuild(strings.Join(args, " "))
		case "RUN":
			var stdout io.Writer
			if quiet {
				stdout = io.Discard
			}
			if err := builder.Run(args, buildah.RunOptions{Stdout: stdout}); err != nil {
				return err
			}
		case "SHELL":
			builder.SetShell(args)
		case "STOPSIGNAL":
			builder.SetStopSignal(strings.Join(args, " "))
		case "USER":
			builder.SetUser(strings.Join(args, " "))
		case "VOLUME":
			builder.AddVolume(strings.Join(args, " "))
		case "WORKINGDIR":
			builder.SetWorkDir(strings.Join(args, " "))
		default:
			return errors.Errorf("illegal command input %q; ignored", command)
		}
	}
	builder.ClearOnBuild()
	return nil
}
