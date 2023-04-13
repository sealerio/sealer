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

package alpha

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/containers/common/pkg/auth"
	digest "github.com/opencontainers/go-digest"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	manifestDescription        = "\n  Creates, modifies, and pushes manifest lists"
	manifestCreateDescription  = "\n  Creates manifest lists."
	manifestAddDescription     = "\n  Adds an image to a manifest list."
	manifestRemoveDescription  = "\n  Removes an image from a manifest list."
	manifestInspectDescription = "\n  Display the contents of a manifest list."
	manifestPushDescription    = "\n  Pushes manifest lists to registries."
	manifestDeleteDescription  = "\n  Remove one or more manifest lists from local storage."
	createManifestOpts         options.ManifestCreateOpts
	addManifestOpts            options.ManifestAddOpts
	removeManifestOpts         options.ManifestRemoveOpts
	deleteManifestOpts         options.ManifestDeleteOpts
	inspectManifestOpts        options.ManifestInspectOpts
	pushManifestOpts           options.PushOptions
)

func NewManifestCmd() *cobra.Command {
	manifestCommand := &cobra.Command{
		Use:   "manifest",
		Short: "manipulate manifest lists",
		Long:  manifestDescription,
		Example: `sealer alpha manifest create localhost/my-manifest
  sealer alpha manifest add localhost/my-manifest localhost/image
  sealer alpha manifest inspect localhost/my-manifest
  sealer alpha manifest push localhost/my-manifest transport:destination
  sealer alpha manifest remove localhost/my-manifest sha256:entryManifestDigest
  sealer alpha manifest delete localhost/my-manifest`,
	}

	manifestCommand.AddCommand(manifestCreateCommand())
	manifestCommand.AddCommand(manifestAddCommand())
	manifestCommand.AddCommand(manifestRemoveCommand())
	manifestCommand.AddCommand(manifestInspectCommand())
	manifestCommand.AddCommand(manifestDeleteCommand())
	manifestCommand.AddCommand(manifestPushCommand())
	return manifestCommand
}

func manifestCreateCommand() *cobra.Command {
	createCommand := &cobra.Command{
		Use:   "create",
		Short: "Create manifest list",
		Long:  manifestCreateDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("at least a name must be specified for the manifest list")
			}
			engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			id, err := engine.CreateManifest(args[0], &createManifestOpts)
			if err != nil {
				return err
			}

			logrus.Infof("successfully create manifest %s with ID %s", args[0], id)
			return nil
		},
		Example: `sealer alpha manifest create mylist:v1.11`,
		Args:    cobra.MinimumNArgs(1),
	}

	return createCommand
}

func manifestAddCommand() *cobra.Command {
	addCommand := &cobra.Command{
		Use:   "add",
		Short: "Add images to a manifest list",
		Long:  manifestAddDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				manifestName = addManifestOpts.TargetName
				imagesToAdd  = args
			)

			// if not set `-t` flag , assume the first one is the manifestName,others is the images need to be added to.
			if manifestName == "" {
				manifestName = args[0]
				imagesToAdd = args[1:]
			}

			engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			return engine.AddToManifest(manifestName, imagesToAdd, &addManifestOpts)
		},
		Example: `sealer alpha manifest add app-amd:v1 app-arm:v1 -t all-in-one:v1
  sealer alpha manifest add mylist:v1.11 image:v1.11-amd64`,
		Args: cobra.MinimumNArgs(1),
	}

	flags := addCommand.Flags()
	flags.StringVar(&addManifestOpts.Os, "os", "", "override the `OS` of the specified image")
	flags.StringVar(&addManifestOpts.Arch, "arch", "", "override the `architecture` of the specified image")
	flags.StringVar(&addManifestOpts.Variant, "variant", "", "override the `variant` of the specified image")
	flags.StringVar(&addManifestOpts.OsVersion, "os-version", "", "override the OS `version` of the specified image")
	flags.StringSliceVar(&addManifestOpts.OsFeatures, "os-features", nil, "override the OS `features` of the specified image")
	flags.StringSliceVar(&addManifestOpts.Annotations, "annotation", nil, "set an `annotation` for the specified image")
	flags.BoolVar(&addManifestOpts.All, "all", false, "add all of the list's images if the image is a list")
	flags.StringVarP(&addManifestOpts.TargetName, "target", "t", "", "target image name,if it is not exist,will create a new one")

	return addCommand
}

func manifestRemoveCommand() *cobra.Command {
	removeCommand := &cobra.Command{
		Use:   "remove",
		Short: "Remove an entry from a manifest list",
		Long:  manifestRemoveDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				name           string
				instanceDigest digest.Digest
			)

			engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			switch len(args) {
			case 0, 1:
				return errors.New("at least a list image and one or more instance digests must be specified ")
			case 2:
				name = args[0]
				if name == "" {
					return fmt.Errorf(`invalid image name "%s" `, args[0])
				}
				instanceSpec := args[1]
				if instanceSpec == "" {
					return fmt.Errorf(`invalid instance "%s" `, args[1])
				}
				d, err := digest.Parse(instanceSpec)
				if err != nil {
					return fmt.Errorf(`invalid instance "%s": %v `, args[1], err)
				}
				instanceDigest = d
			default:
				return errors.New("at least two arguments are necessary: list and digest of instance to remove from list ")
			}

			return engine.RemoveFromManifest(name, instanceDigest, &removeManifestOpts)
		},
		Example: `sealer alpha manifest remove mylist:v1.11 sha256:15352d97781ffdf357bf3459c037be3efac4133dc9070c2dce7eca7c05c3e736`,
		Args:    cobra.MinimumNArgs(2),
	}

	return removeCommand
}

func manifestInspectCommand() *cobra.Command {
	inspectCommand := &cobra.Command{
		Use:   "inspect",
		Short: "Display the contents of a manifest list",
		Long:  manifestInspectDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				name string
			)
			switch len(args) {
			case 0:
				return errors.New("at least a source list ID must be specified")
			case 1:
				name = args[0]
				if name == "" {
					return fmt.Errorf(`invalid manifest name "%s" `, name)
				}
			default:
				return errors.New("only one argument is necessary for inspect: an manifest name")
			}

			engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			schema2List, err := engine.InspectManifest(name, &inspectManifestOpts)
			if err != nil {
				return err
			}

			b, err := json.MarshalIndent(schema2List, "", "    ")
			if err != nil {
				return err
			}

			fmt.Println(string(b))
			return nil
		},
		Example: `sealer alpha manifest inspect mylist:v1.11`,
		Args:    cobra.MinimumNArgs(1),
	}
	return inspectCommand
}

func manifestDeleteCommand() *cobra.Command {
	deleteCommand := &cobra.Command{
		Use:   "delete",
		Short: "Delete manifest list",
		Long:  manifestDeleteDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			return engine.DeleteManifests(args, &deleteManifestOpts)
		},
		Example: `sealer alpha manifest delete mylist:v1.11`,
		Args:    cobra.MinimumNArgs(1),
	}
	return deleteCommand
}

func manifestPushCommand() *cobra.Command {
	pushCommand := &cobra.Command{
		Use:   "push",
		Short: "Push a manifest list to a registry",
		Long:  manifestPushDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.CheckAuthFile(pushManifestOpts.Authfile); err != nil {
				return err
			}

			var (
				name, destSpec string
			)

			switch len(args) {
			case 0:
				return errors.New("at least a source list ID must be specified ")
			case 1:
				name = args[0]
				destSpec = args[0]
			case 2:
				name = args[0]
				destSpec = args[1]
				if name == "" {
					return fmt.Errorf(`invalid manifest name "%s"`, name)
				}
				if destSpec == "" {
					return fmt.Errorf(`invalid image name "%s"`, destSpec)
				}
			default:
				return errors.New("need one Or two arguments are necessary to push: source and destination ")
			}

			engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			return engine.PushManifest(name, destSpec, &pushManifestOpts)
		},
		Example: `sealer alpha manifest push mylist:v1.11 transport:imageName`,
		Args:    cobra.MinimumNArgs(1),
	}

	flags := pushCommand.Flags()
	flags.BoolVar(&pushManifestOpts.Rm, "rm", false, "remove the manifest list if push succeeds")
	flags.BoolVar(&pushManifestOpts.All, "all", false, "also push the images in the list")
	flags.StringVar(&pushManifestOpts.Authfile, "authfile", auth.GetDefaultAuthFile(), "path of the authentication file. Use REGISTRY_AUTH_FILE environment variable to override")
	flags.StringVar(&pushManifestOpts.CertDir, "cert-dir", "", "use certificates at the specified path to access the registry")
	flags.StringVarP(&pushManifestOpts.Format, "format", "f", "", "manifest type (oci or v2s2) to attempt to use when pushing the manifest list (default is manifest type of source)")
	flags.BoolVar(&pushManifestOpts.SkipTLSVerify, "skip-tls-verify", false, "default is requiring HTTPS and verify certificates when accessing the registry.")
	flags.BoolVarP(&pushManifestOpts.Quiet, "quiet", "q", false, "don't output progress information when pushing lists")

	return pushCommand
}
