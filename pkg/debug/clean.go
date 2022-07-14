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

package debug

import (
	"bytes"
	"context"
	"fmt"
	"strings"

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

// CleanOptions holds the options for an invocation of debug clean.
type CleanOptions struct {
	PodName       string
	Namespace     string
	ContainerName string
}

// Cleaner cleans the debug containers and pods.
type Cleaner struct {
	*CleanOptions

	AdminKubeConfigPath string

	stdin bool
	tty   bool

	genericclioptions.IOStreams
}

func NewDebugCleanOptions() *CleanOptions {
	return &CleanOptions{}
}

func NewDebugCleaner() *Cleaner {
	return &Cleaner{
		CleanOptions: NewDebugCleanOptions(),
	}
}

// CompleteAndVerifyOptions completes and verifies DebugCleanOptions.
func (cleaner *Cleaner) CompleteAndVerifyOptions(args []string) error {
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
func (cleaner *Cleaner) Run() error {
	ctx := context.Background()

	// get the rest config
	restConfig, err := clientcmd.BuildConfigFromFlags("", cleaner.AdminKubeConfigPath)
	if err != nil {
		return errors.Wrapf(err, "failed to get rest config from file %s", cleaner.AdminKubeConfigPath)
	}
	if err := SetKubernetesDefaults(restConfig); err != nil {
		return err
	}

	// get the kube client set
	kubeClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to create kubernetes client from file %s", cleaner.AdminKubeConfigPath)
	}

	if strings.HasPrefix(cleaner.PodName, NodeDebugPrefix) {
		return cleaner.RemovePod(ctx, kubeClientSet.CoreV1())
	}

	return cleaner.ExitEphemeralContainer(restConfig)
}

// RemovePod removes the debug pods.
func (cleaner *Cleaner) RemovePod(ctx context.Context, kubeClientCorev1 corev1client.CoreV1Interface) error {
	if kubeClientCorev1 == nil {
		return fmt.Errorf("clean must need a kubernetes client")
	}

	return kubeClientCorev1.Pods(cleaner.Namespace).Delete(ctx, cleaner.PodName, metav1.DeleteOptions{})
}

// ExitEphemeralContainer exits the ephemeral containers
// and the ephemeral container's status will become terminated.
func (cleaner *Cleaner) ExitEphemeralContainer(config *restclient.Config) error {
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
		Container: cleaner.ContainerName,
		Stdin:     cleaner.stdin,
		Stdout:    cleaner.Out != nil,
		Stderr:    cleaner.ErrOut != nil,
		TTY:       cleaner.tty,
	}, scheme.ParameterCodec)

	connect := &Connector{
		Config:    config,
		IOStreams: cleaner.IOStreams,
		TTY:       cleaner.tty,
	}

	if err := connect.DoConnect("POST", req.URL(), nil); err != nil {
		return err
	}

	return nil
}

func (cleaner *Cleaner) initExitEpheContainerOpts() {
	var stdout, stderr bytes.Buffer
	stdin := bytes.NewBuffer([]byte("exit\n"))
	cleaner.In = stdin
	cleaner.Out = &stdout
	cleaner.ErrOut = &stderr
	cleaner.stdin = true
	cleaner.tty = true
}
