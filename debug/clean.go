package debug

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/alibaba/sealer/common"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type CleanOptions struct {
	PodName				string
	Namespace			string
	KubeClientCorev1	corev1client.CoreV1Interface

	Config				*restclient.Config
	ContainerName		string
	Stdin 				bool
	TTY					bool

	genericclioptions.IOStreams
}

func NewCleanOptions() *CleanOptions {
	return &CleanOptions{}
}

func NewDebugClean() *cobra.Command {
	cleanOptions := NewCleanOptions()

	cmd := &cobra.Command{
		Use:     "clean",
		Short:   "clean the debug container od pod",
		Long:    "",
		Example: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cleanOptions.CompleteAndVerify(cmd, args); err != nil {
				return err
			}
			if err := cleanOptions.Run(cmd); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func (cleanOpts *CleanOptions) CompleteAndVerify(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("debug ID is required for clean")
	}

	ss := strings.Split(args[0], DEBUG_ID_FS)
	if len(ss) < 3 {
		return fmt.Errorf("invaild debug ID")
	}

	cleanOpts.Namespace = ss[2]
	cleanOpts.PodName = ss[1]
	cleanOpts.ContainerName = ss[0]

	return nil
}

func (cleanOpts *CleanOptions) Run(cmd *cobra.Command) error {
	ctx := context.Background()

	// Diff: between trident and sealer
	// adminKubeConfigPath := config.AdminKubeConfPath
	adminKubeConfigPath := common.KubeAdminConf

	// get the rest config
	restConfig, err := clientcmd.BuildConfigFromFlags("", adminKubeConfigPath)
	if err != nil {
		return errors.Wrapf(err, "failed to get rest config from file %s", adminKubeConfigPath)
	}
	setKubernetesDefaults(restConfig)
	cleanOpts.Config = restConfig

	// get the kube client set
	kubeClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to create kubernetes client from file %s", adminKubeConfigPath)
	}
	cleanOpts.KubeClientCorev1 = kubeClientSet.CoreV1()

	if strings.HasPrefix(cleanOpts.PodName, NODE_DEBUG_PREFIX) {
		return cleanOpts.RemovePod(ctx)
	} else {
		return cleanOpts.ExitEphemeralContainer(ctx)
	}

	return nil
}

// RemovePod removes the connect pod
func (cleanOpts *CleanOptions) RemovePod(ctx context.Context) error {
	if cleanOpts.KubeClientCorev1 == nil {
		return fmt.Errorf("clean must need a kubernetes client")
	}

	return cleanOpts.KubeClientCorev1.Pods(cleanOpts.Namespace).Delete(ctx, cleanOpts.PodName, metav1.DeleteOptions{})
}

// ExitEphemeralContainer exits the ephemeral container and the ephemeral container's status
// will become terminated.
func (cleanOpts *CleanOptions) ExitEphemeralContainer(ctx context.Context) error {
	restClient, err := restclient.RESTClientFor(cleanOpts.Config)
	if err != nil {
		return err
	}

	cleanOpts.initExitEpheContainerOpts()

	req := restClient.Post().
		Resource("pods").
		Name(cleanOpts.PodName).
		Namespace(cleanOpts.Namespace).
		SubResource("attach")
	req.VersionedParams(&corev1.PodAttachOptions{
		Container: 		cleanOpts.ContainerName,
		Stdin: 			cleanOpts.Stdin,
		Stdout: 		cleanOpts.Out != nil,
		Stderr: 		cleanOpts.ErrOut != nil,
		TTY:			cleanOpts.TTY,
	}, scheme.ParameterCodec)

	connect := &ConnectOptions{
		Config:		cleanOpts.Config,
		IOStreams:	cleanOpts.IOStreams,
		TTY:		cleanOpts.TTY,
	}

	if err := connect.DoConnect("POST", req.URL(), nil); err != nil {
		return err
	}

	return nil
}

func (cleanOpts *CleanOptions) initExitEpheContainerOpts() {
	var stdout, stderr bytes.Buffer
	stdin := bytes.NewBuffer([]byte("exit\n"))
	cleanOpts.In = stdin
	cleanOpts.Out = &stdout
	cleanOpts.ErrOut = &stderr
	cleanOpts.Stdin = true
	cleanOpts.TTY = true
}