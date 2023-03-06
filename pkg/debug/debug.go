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
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	watchtools "k8s.io/client-go/tools/watch"

	"github.com/sealerio/sealer/pkg/debug/clusterinfo"
)

const (
	NodeDebugPrefix = "node-debugger"
	PodDebugPrefix  = "pod-debugger"

	TypeDebugNode = "node"
	TypeDebugPod  = "pod"

	FSDebugID = "."
)

// DebuggerOptions holds the options for an invocation of debug.
type DebuggerOptions struct {
	Type       string // debug pod or node
	TargetName string // pod/node name to be debugged

	Image       string // debug container/pod image name
	Env         []corev1.EnvVar
	Interactive bool     // -i
	TTY         bool     // -t
	Command     []string // after --
	CheckList   []string // check network、volume etc

	DebugContainerName string // debug container name
	Namespace          string // kubernetes namespace
	PullPolicy         string

	AdminKubeConfigPath string

	// Type is container
	TargetContainer string // target container to share the namespace
}

type Debugger struct {
	*DebuggerOptions
	Motd string

	kubeClientCorev1 corev1client.CoreV1Interface

	genericclioptions.IOStreams
}

// NewDebugOptions returns a DebugOptions initialized with default values.
func NewDebugOptions() *DebuggerOptions {
	return &DebuggerOptions{
		Command: []string{},

		Namespace:  corev1.NamespaceDefault,
		PullPolicy: string(corev1.PullIfNotPresent),
	}
}

func NewDebugger(options *DebuggerOptions) *Debugger {
	return &Debugger{
		DebuggerOptions: options,
		IOStreams: genericclioptions.IOStreams{
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
	}
}

// CompleteAndVerifyOptions completes and verifies DebugOptions.
func (debugger *Debugger) CompleteAndVerifyOptions(cmd *cobra.Command, args []string, imager ImagesManagement) error {
	// args
	debugger.TargetName = args[0]
	argsLen := cmd.ArgsLenAtDash()

	if argsLen == -1 && len(args) > 1 {
		debugger.Command = args[1:]
	}

	if argsLen > 0 && len(args) > argsLen {
		debugger.Command = args[argsLen:]
	}

	if len(debugger.Image) == 0 {
		image, err := imager.GetDefaultImage()
		if err != nil {
			return err
		}

		debugger.Image = image
	}

	if len(debugger.Image) > 0 && !reference.ReferenceRegexp.MatchString(debugger.Image) {
		return fmt.Errorf("invalid image name %q: %v", debugger.Image, reference.ErrReferenceInvalidFormat)
	}

	// stdin/tty
	if debugger.TTY || debugger.Interactive {
		debugger.In = os.Stdin
		debugger.Interactive = true
	}

	// env
	envStrings, err := cmd.Flags().GetStringToString("env")
	if err != nil {
		return fmt.Errorf("error getting env flag: %v", err)
	}
	for k, v := range envStrings {
		debugger.Env = append(debugger.Env, corev1.EnvVar{Name: k, Value: v})
	}

	// PullPolicy
	if strings.EqualFold(debugger.PullPolicy, string(corev1.PullAlways)) {
		debugger.PullPolicy = string(corev1.PullAlways)
	}

	if strings.EqualFold(debugger.PullPolicy, string(corev1.PullIfNotPresent)) {
		debugger.PullPolicy = string(corev1.PullIfNotPresent)
	}

	if strings.EqualFold(debugger.PullPolicy, string(corev1.PullNever)) {
		debugger.PullPolicy = string(corev1.PullNever)
	}

	// checklist: add check items into env
	debugger.Env = append(debugger.Env, corev1.EnvVar{
		Name:  "CHECK_LIST",
		Value: strings.Join(debugger.CheckList, " "),
	})

	return nil
}

// Run generates a debug pod/node and attach to it according to command flag.
func (debugger *Debugger) Run() (string, error) {
	ctx := context.Background()

	// get the rest config
	restConfig, err := clientcmd.BuildConfigFromFlags("", debugger.AdminKubeConfigPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get rest config from file %s", debugger.AdminKubeConfigPath)
	}
	if err := SetKubernetesDefaults(restConfig); err != nil {
		return "", err
	}

	// get the kube client set
	kubeClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create kubernetes client from file %s", debugger.AdminKubeConfigPath)
	}
	debugger.kubeClientCorev1 = kubeClientSet.CoreV1()

	var (
		debugPod *corev1.Pod
		errDebug error
	)

	// generate a debug container or pod
	if debugger.Type == TypeDebugNode {
		debugPod, errDebug = debugger.DebugNode(ctx)
	} else {
		debugPod, errDebug = debugger.DebugPod(ctx)
	}

	if errDebug != nil {
		return "", errDebug
	}

	// will only create debug container but will not to connect it
	if len(debugger.Command) == 0 && !debugger.TTY {
		return debugger.getDebugID(debugPod), nil
	}

	// clean the debugger container/pod
	clean := &Cleaner{
		CleanOptions: &CleanOptions{
			Namespace:     debugPod.Namespace,
			PodName:       debugPod.Name,
			ContainerName: debugger.DebugContainerName,
		},
	}

	if errCon := debugger.connectPod(ctx, debugPod, restConfig); errCon != nil {
		// There is no error handling because they are the default clean actions.
		// Even if it returns an error, we should not return the error to user.
		if debugger.Type == TypeDebugNode {
			_ = clean.RemovePod(ctx, debugger.kubeClientCorev1)
		} else {
			_ = clean.ExitEphemeralContainer(restConfig)
		}

		return "", errCon
	}

	// It is the same as before.
	if debugger.Type == TypeDebugNode {
		_ = clean.RemovePod(ctx, debugger.kubeClientCorev1)
	} else {
		_ = clean.ExitEphemeralContainer(restConfig)
	}

	return "", nil
}

// addClusterInfoIntoEnv adds the cluster infos into DebugOptions.Env
func (debugger *Debugger) addClusterInfoIntoEnv(ctx context.Context) error {
	podsIPList, err := clusterinfo.GetPodsIP(ctx, debugger.kubeClientCorev1, debugger.Namespace)
	if err != nil {
		return err
	}
	debugger.Env = append(debugger.Env, corev1.EnvVar{
		Name:  "POD_IP_LIST",
		Value: strings.Join(podsIPList, " "),
	})

	nodesIPList, err := clusterinfo.GetNodesIP(ctx, debugger.kubeClientCorev1)
	if err != nil {
		return err
	}
	debugger.Env = append(debugger.Env, corev1.EnvVar{
		Name:  "NODE_IP_LIST",
		Value: strings.Join(nodesIPList, " "),
	})

	dnsSVCName, dnsSVCIP, dnsEndpointsIPs, err := clusterinfo.GetDNSServiceAll(ctx, debugger.kubeClientCorev1)
	if err != nil {
		return err
	}
	debugger.Env = append(debugger.Env,
		corev1.EnvVar{
			Name:  "KUBE_DNS_SERVICE_NAME",
			Value: dnsSVCName,
		},
		corev1.EnvVar{
			Name:  "KUBE_DNS_SERVICE_IP",
			Value: dnsSVCIP,
		},
		corev1.EnvVar{
			Name:  "KUBE_DNS_ENDPOINTS_IPS",
			Value: strings.Join(dnsEndpointsIPs, " "),
		},
	)

	return nil
}

func (debugger *Debugger) connectPod(ctx context.Context, debugPod *corev1.Pod, restConfig *rest.Config) error {
	// wait the debug container(ephemeral container) running
	debugPodRun, err := WaitForContainer(ctx, debugger.kubeClientCorev1, debugPod.Namespace, debugPod.Name, debugger.DebugContainerName)
	if err != nil {
		return err
	}

	status := GetContainerStatusByName(debugPodRun, debugger.DebugContainerName)
	if status == nil {
		return fmt.Errorf("error getting container status of container name %s", debugger.DebugContainerName)
	}

	if status.State.Terminated != nil {
		return fmt.Errorf("debug container %s terminated", debugger.DebugContainerName)
	}

	// begin attaching to debug container(ephemeral container)
	connectOpts := &Connector{
		NameSpace:     debugPodRun.Namespace,
		Pod:           debugPodRun,
		Command:       debugger.Command,
		ContainerName: debugger.DebugContainerName,
		Stdin:         debugger.Interactive,
		TTY:           debugger.TTY,
		IOStreams:     debugger.IOStreams,
		Config:        restConfig,
		Motd:          debugger.Motd,
	}

	if err := connectOpts.Connect(); err != nil {
		return err
	}

	return nil
}

// getDebugID returns the debug ID that consists of namespace, pod name, container name
func (debugger *Debugger) getDebugID(pod *corev1.Pod) string {
	return debugger.DebugContainerName + FSDebugID + pod.Name + FSDebugID + debugger.Namespace
}

// SetKubernetesDefaults sets default values on the provided client config for accessing the
// Kubernetes API or returns an error if any of the defaults are impossible or invalid.
func SetKubernetesDefaults(config *rest.Config) error {
	if config.GroupVersion == nil {
		config.GroupVersion = &corev1.SchemeGroupVersion
	}

	if config.APIPath == "" {
		config.APIPath = "/api"
	}

	if config.NegotiatedSerializer == nil {
		config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
		//restConfig.NegotiatedSerializer = scheme.Codecs
	}

	return rest.SetKubernetesDefaults(config)
}

// WaitForContainer watches the given pod until the container is running or terminated.
func WaitForContainer(ctx context.Context, client corev1client.PodsGetter, namespace, podName, containerName string) (*corev1.Pod, error) {
	ctx, cancel := watchtools.ContextWithOptionalTimeout(ctx, 5*time.Second)
	defer cancel()

	// register the watcher and lister
	fieldSelector := fields.OneTermEqualSelector("metadata.name", podName).String()
	listAndWatch := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return client.Pods(namespace).List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return client.Pods(namespace).Watch(ctx, options)
		},
	}

	// waiting sync
	event, err := watchtools.UntilWithSync(ctx, listAndWatch, &corev1.Pod{}, nil, func(event watch.Event) (bool, error) {
		switch event.Type {
		case watch.Deleted:
			return false, apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "")
		}

		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			return false, fmt.Errorf("watch did not return a pod: %v", event.Object)
		}

		status := GetContainerStatusByName(pod, containerName)
		if status == nil {
			return false, nil
		}

		if status.State.Waiting != nil && status.State.Waiting.Reason == "ImagePullBackOff" {
			return false, fmt.Errorf("failed to pull image: (%s: %s)", status.State.Waiting.Reason, status.State.Waiting.Message)
		}

		if status.State.Running != nil || status.State.Terminated != nil {
			return true, nil
		}

		return false, nil
	})

	if event != nil {
		return event.Object.(*corev1.Pod), err
	}

	return nil, err
}

// GetContainerStatusByName returns the container status by the containerName.
func GetContainerStatusByName(pod *corev1.Pod, containerName string) *corev1.ContainerStatus {
	allContainerStatus := [][]corev1.ContainerStatus{pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses, pod.Status.EphemeralContainerStatuses}

	for _, statusSlice := range allContainerStatus {
		for _, status := range statusSlice {
			if status.Name == containerName {
				return &status
			}
		}
	}

	return nil
}

// ContainerNameToRef returns the container names in pod.
func ContainerNameToRef(pod *corev1.Pod) map[string]*corev1.Container {
	names := map[string]*corev1.Container{}

	for i := range pod.Spec.Containers {
		ref := &pod.Spec.Containers[i]
		names[ref.Name] = ref
	}

	for i := range pod.Spec.InitContainers {
		ref := &pod.Spec.InitContainers[i]
		names[ref.Name] = ref
	}

	for i := range pod.Spec.EphemeralContainers {
		ref := (*corev1.Container)(&pod.Spec.EphemeralContainers[i].EphemeralContainerCommon)
		names[ref.Name] = ref
	}

	return names
}
