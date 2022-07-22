package buildah

import (
	"fmt"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/imagebuildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	buildahutil "github.com/containers/buildah/pkg/util"
	"github.com/containers/buildah/util"
	"github.com/containers/common/pkg/auth"
	"github.com/pkg/errors"
	"github.com/sealerio/sealer/pkg/image_adaptor/common"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Adaptor struct {
	*buildahcli.BudResults
	*buildahcli.LayerResults
	*buildahcli.FromAndBudResults
	*buildahcli.NameSpaceResults
	*buildahcli.UserNSResults
	*cobra.Command
}

func (adaptor *Adaptor) Build(sealerBuildFlags *common.BuildFlags, inputArgs []string) error {
	flags := adaptor.Flags()
	buildFlags := buildahcli.GetBudFlags(adaptor.BudResults)
	buildFlags.StringVar(&adaptor.Runtime, "runtime", util.Runtime(), "`path` to an alternate runtime. Use BUILDAH_RUNTIME environment variable to override.")

	layerFlags := buildahcli.GetLayerFlags(adaptor.LayerResults)
	fromAndBudFlags, err := buildahcli.GetFromAndBudFlags(adaptor.FromAndBudResults, adaptor.UserNSResults, adaptor.NameSpaceResults)
	if err != nil {
		return fmt.Errorf("failed to setup From and Build flags: %v", err)
	}

	flags.AddFlagSet(&buildFlags)
	flags.AddFlagSet(&layerFlags)
	flags.AddFlagSet(&fromAndBudFlags)
	flags.SetNormalizeFunc(buildahcli.AliasFlags)

	err = adaptor.migrateFlag2Buildah(sealerBuildFlags)
	if err != nil {
		return err
	}

	output := ""
	cleanTmpFile := false
	tags := []string{}
	if adaptor.Flag("tag").Changed {
		tags = adaptor.Tag
		if len(tags) > 0 {
			output = tags[0]
			tags = tags[1:]
		}
		if adaptor.Flag("manifest").Changed {
			for _, tag := range tags {
				if tag == adaptor.Manifest {
					return errors.New("the same name must not be specified for both '--tag' and '--manifest'")
				}
			}
		}
	}

	if err := auth.CheckAuthFile(adaptor.Authfile); err != nil {
		return err
	}
	adaptor.Authfile, cleanTmpFile =
		buildahutil.MirrorToTempFileIfPathIsDescriptor(adaptor.Authfile)
	if cleanTmpFile {
		defer os.Remove(adaptor.Authfile)
	}

	// Allow for --pull, --pull=true, --pull=false, --pull=never, --pull=always
	// --pull-always and --pull-never.  The --pull-never and --pull-always options
	// will not be documented.
	pullPolicy := define.PullIfMissing
	if strings.EqualFold(strings.TrimSpace(adaptor.Pull), "true") {
		pullPolicy = define.PullIfNewer
	}
	if adaptor.PullAlways || strings.EqualFold(strings.TrimSpace(adaptor.Pull), "always") {
		pullPolicy = define.PullAlways
	}
	if adaptor.PullNever || strings.EqualFold(strings.TrimSpace(adaptor.Pull), "never") {
		pullPolicy = define.PullNever
	}
	logrus.Debugf("Pull Policy for pull [%v]", pullPolicy)

	kubefiles := getKubefiles(adaptor.File)
	format, err := getFormat(adaptor.Format)
	if err != nil {
		return err
	}

	layers := buildahcli.UseLayers()
	if adaptor.Flag("layers").Changed {
		layers = adaptor.Layers
	}

	contextDir := ""
	cliArgs := inputArgs

	// Nothing provided, we assume the current working directory as build
	// context
	if len(cliArgs) == 0 {
		contextDir, err = os.Getwd()
		if err != nil {
			return errors.Wrapf(err, "unable to choose current working directory as build context")
		}
	} else {
		// It was local.  Use it as is.
		absDir, err := filepath.Abs(cliArgs[0])
		if err != nil {
			return errors.Wrapf(err, "error determining path to directory")
		}
		contextDir = absDir
	}

	cliArgs = tail(cliArgs)

	if err := buildahcli.VerifyFlagsArgsOrder(cliArgs); err != nil {
		return err
	}

	if len(kubefiles) == 0 {
		kubefile, err := DiscoverKubefile(contextDir)
		if err != nil {
			return err
		}
		kubefiles = append(kubefiles, kubefile)
	}

	contextDir, err = filepath.EvalSymlinks(contextDir)
	if err != nil {
		return errors.Wrapf(err, "error evaluating symlinks in build context path")
	}

	var stdin io.Reader
	if adaptor.Stdin {
		stdin = os.Stdin
	}
	var stdout, stderr, reporter *os.File = os.Stdout, os.Stderr, os.Stderr
	if adaptor.Flag("logfile").Changed {
		f, err := os.OpenFile(adaptor.Logfile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err != nil {
			return errors.Errorf("error opening logfile %q: %v", adaptor.Logfile, err)
		}
		defer f.Close()
		logrus.SetOutput(f)
		stdout = f
		stderr = f
		reporter = f
	}

	store, err := getStore(adaptor.Command)
	if err != nil {
		return err
	}

	systemContext, err := parse.SystemContextFromOptions(adaptor.Command)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	isolation, err := parse.IsolationOption(adaptor.Isolation)
	if err != nil {
		return err
	}

	runtimeFlags := []string{}
	for _, arg := range adaptor.RuntimeFlags {
		runtimeFlags = append(runtimeFlags, "--"+arg)
	}

	commonOpts, err := parse.CommonBuildOptions(adaptor.Command)
	if err != nil {
		return err
	}

	namespaceOptions, networkPolicy, err := parse.NamespaceOptions(adaptor.Command)
	if err != nil {
		return err
	}

	usernsOption, idmappingOptions, err := parse.IDMappingOptions(adaptor.Command, isolation)
	if err != nil {
		return errors.Wrapf(err, "error parsing ID mapping options")
	}
	namespaceOptions.AddOrReplace(usernsOption...)

	platforms, err := parse.PlatformsFromOptions(adaptor.Command)
	if err != nil {
		return err
	}

	decConfig, err := getDecryptConfig(adaptor.DecryptionKeys)
	if err != nil {
		return errors.Wrapf(err, "unable to obtain decrypt config")
	}

	var excludes []string
	if adaptor.IgnoreFile != "" {
		if excludes, _, err = parse.ContainerIgnoreFile(contextDir, adaptor.IgnoreFile); err != nil {
			return err
		}
	}

	var timestamp *time.Time
	if adaptor.Command.Flag("timestamp").Changed {
		t := time.Unix(adaptor.Timestamp, 0).UTC()
		timestamp = &t
	}

	compression := define.Gzip
	if adaptor.DisableCompression {
		compression = define.Uncompressed
	}

	options := define.BuildOptions{
		AddCapabilities: adaptor.CapAdd,
		AdditionalTags:  tags,
		AllPlatforms:    adaptor.AllPlatforms,
		Annotations:     adaptor.Annotation,
		Architecture:    systemContext.ArchitectureChoice,
		//Args:                    args,
		BlobDirectory:           adaptor.BlobCache,
		CNIConfigDir:            adaptor.CNIConfigDir,
		CNIPluginPath:           adaptor.CNIPlugInPath,
		CommonBuildOpts:         commonOpts,
		Compression:             compression,
		ConfigureNetwork:        networkPolicy,
		ContextDirectory:        contextDir,
		DefaultMountsFilePath:   "",
		Devices:                 adaptor.Devices,
		DropCapabilities:        adaptor.CapDrop,
		Err:                     stderr,
		ForceRmIntermediateCtrs: adaptor.ForceRm,
		From:                    adaptor.From,
		IDMappingOptions:        idmappingOptions,
		IIDFile:                 adaptor.Iidfile,
		In:                      stdin,
		Isolation:               isolation,
		IgnoreFile:              adaptor.IgnoreFile,
		Labels:                  adaptor.Label,
		Layers:                  layers,
		LogRusage:               adaptor.LogRusage,
		Manifest:                adaptor.Manifest,
		MaxPullPushRetries:      maxPullPushRetries,
		NamespaceOptions:        namespaceOptions,
		NoCache:                 adaptor.NoCache,
		OS:                      systemContext.OSChoice,
		Out:                     stdout,
		Output:                  output,
		OutputFormat:            format,
		PullPolicy:              pullPolicy,
		PullPushRetryDelay:      pullPushRetryDelay,
		Quiet:                   adaptor.Quiet,
		RemoveIntermediateCtrs:  adaptor.Rm,
		ReportWriter:            reporter,
		Runtime:                 adaptor.Runtime,
		RuntimeArgs:             runtimeFlags,
		RusageLogFile:           adaptor.RusageLogFile,
		SignBy:                  adaptor.SignBy,
		SignaturePolicyPath:     adaptor.SignaturePolicy,
		Squash:                  adaptor.Squash,
		SystemContext:           systemContext,
		Target:                  adaptor.Target,
		TransientMounts:         adaptor.Volumes,
		OciDecryptConfig:        decConfig,
		Jobs:                    &adaptor.Jobs,
		Excludes:                excludes,
		Timestamp:               timestamp,
		Platforms:               platforms,
		UnsetEnvs:               adaptor.UnsetEnvs,
	}

	logrus.Infof("final options is: %+v", options)

	if adaptor.Quiet {
		options.ReportWriter = ioutil.Discard
	}

	id, ref, err := imagebuildah.BuildDockerfiles(getContext(), store, options, kubefiles...)
	if err == nil && options.Manifest != "" {
		logrus.Debugf("manifest list id = %q, ref = %q", id, ref.String())
	}
	return err
}

func getKubefiles(files []string) []string {
	var kubefiles []string
	for _, f := range files {
		if f == "-" {
			kubefiles = append(kubefiles, "/dev/stdin")
		} else {
			kubefiles = append(kubefiles, f)
		}
	}
	return kubefiles
}

// this function aims to set buildah configuration based on sealer image_adaptor flags.
func (adaptor *Adaptor) migrateFlag2Buildah(sealerBuildFlags *common.BuildFlags) error {
	flags := adaptor.Flags()
	// image_adaptor cache related flags
	// cache is enabled when "layers" is true & "no-cache" is false
	_ = flags.Set("layers", "true")
	adaptor.Layers = !sealerBuildFlags.NoCache
	adaptor.NoCache = sealerBuildFlags.NoCache
	// tags. Like -t kubernetes:v1.16
	_ = flags.Set("tag", strings.Join(sealerBuildFlags.Tags, ","))
	adaptor.Tag = sealerBuildFlags.Tags

	// Hardcoded for network configuration.
	// check parse.NamespaceOptions for detailed logic.
	// this network setup for stage container, especially for RUN wget and so on.
	// so I think we can set as host network.
	err := flags.Set("network", "host")
	if err != nil {
		return err
	}

	// set platform to the flags in buildah
	// check the detail in parse.PlatformsFromOptions
	return flags.Set("platform", sealerBuildFlags.Platform)
}

func prepareContainerImages() {

}
