package debug

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/alibaba/sealer/common"

	"github.com/spf13/cobra"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// DebugCleanOptions holds the the options for an invocation of debug clean.
type DebugCleanOptions struct {
	PodName				string
	Namespace			string
	ContainerName		string
}

// DebugCleaner cleans the debug containers and pods.
type DebugCleaner struct {
	*DebugCleanOptions

	AdminKubeConfigPath	string

	stdin 				bool
	tty					bool

	genericclioptions.IOStreams
}

func NewDebugCleanOptions() *DebugCleanOptions {
	return &DebugCleanOptions{}
}

func NewDebugCleaner() *DebugCleaner {
	return &DebugCleaner{
		DebugCleanOptions: NewDebugCleanOptions(),
	}
}

func NewDebugCleanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clean",
		Short:   "Clean the debug container od pod",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cleaner := NewDebugCleaner()
			cleaner.AdminKubeConfigPath = common.KubeAdminConf

			if err := cleaner.CompleteAndVerifyOptions(args); err != nil {
				return err
			}
			if err := cleaner.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

// CompleteAndVerifyOptions completes and verifies DebugCleanOptions.
func (cleaner *DebugCleaner) CompleteAndVerifyOptions(args []string) error {
	ss := strings.Split(args[0], FSDebugID)
	if len(ss) < 3 {
		return fmt.Errorf("invaild debug ID")
	}

	cleaner.Namespace = ss[2]
	cleaner.PodName = ss[1]
	cleaner.ContainerName = ss[0]

	return nil
}

// Run removes debug pods or exits debug containers.
func (cleaner *DebugCleaner) Run() error {
	ctx := context.Background()

	// get the rest config
	restConfig, err := clientcmd.BuildConfigFromFlags("", cleaner.AdminKubeConfigPath)
	if err != nil {
		return errors.Wrapf(err, "failed to get rest config from file %s", cleaner.AdminKubeConfigPath)
	}
	SetKubernetesDefaults(restConfig)

	// get the kube client set
	kubeClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to create kubernetes client from file %s", cleaner.AdminKubeConfigPath)
	}

	if strings.HasPrefix(cleaner.PodName, NodeDebugPrefix) {
		return cleaner.RemovePod(ctx, kubeClientSet.CoreV1())
	} else {
		return cleaner.ExitEphemeralContainer(restConfig)
	}

	return nil
}

// RemovePod removes the debug pods.
func (cleaner *DebugCleaner) RemovePod(ctx context.Context, kubeClientCorev1 corev1client.CoreV1Interface) error {
	if kubeClientCorev1 == nil {
		return fmt.Errorf("clean must need a kubernetes client")
	}

	return kubeClientCorev1.Pods(cleaner.Namespace).Delete(ctx, cleaner.PodName, metav1.DeleteOptions{})
}

// ExitEphemeralContainer exits the ephemeral containers
// and the ephemeral container's status will become terminated.
func (cleaner *DebugCleaner) ExitEphemeralContainer(config *restclient.Config) error {
	restClient, err := restclient.RESTClientFor(config)
	if err != nil {
		return err
	}

	cleaner.initExitEpheContainerOpts()

	req := restClient.Post().
		Resource("pods").
		Name(cleaner.PodName).
		Namespace(cleaner.Namespace).
		SubResource("attach")
	req.VersionedParams(&corev1.PodAttachOptions{
		Container: 		cleaner.ContainerName,
		Stdin: 			cleaner.stdin,
		Stdout: 		cleaner.Out != nil,
		Stderr: 		cleaner.ErrOut != nil,
		TTY:			cleaner.tty,
	}, scheme.ParameterCodec)

	connect := &Connector{
		Config:		config,
		IOStreams:	cleaner.IOStreams,
		TTY:		cleaner.tty,
	}

	if err := connect.DoConnect("POST", req.URL(), nil); err != nil {
		return err
	}

	return nil
}

func (cleaner *DebugCleaner) initExitEpheContainerOpts() {
	var stdout, stderr bytes.Buffer
	stdin := bytes.NewBuffer([]byte("exit\n"))
	cleaner.In = stdin
	cleaner.Out = &stdout
	cleaner.ErrOut = &stderr
	cleaner.stdin = true
	cleaner.tty = true
}