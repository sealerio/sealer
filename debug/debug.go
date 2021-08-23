package debug

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/debug/clusterinfo"

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
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	watchtools "k8s.io/client-go/tools/watch"
)

const DEBUG_ID_FS = "."

// DebugOptions holds the options for an invocation of debug.
type DebugOptions struct {
	Type				string		// debug pod or node
	TargetName			string		// pod/node name to be debugged

	Image				string		// debug container/pod image name
	Env					[]corev1.EnvVar
	Interactive			bool		// -i
	TTY					bool		// -t
	Command				[]string	// after --
	CheckList           []string	// check network、volume etc

	DebugContainerName	string		// debug container name
	Namespace			string		// kubernetes namespace
	PullPolicy			corev1.PullPolicy

	kubeClientCorev1	corev1client.CoreV1Interface
	kubeClientRBACV1	rbacv1client.RbacV1Interface

	genericclioptions.IOStreams
}

// NewDebugOptions returns a DebugOptions initialized with default values.
func NewDebugOptions() *DebugOptions {
	return &DebugOptions{
		Command:		[]string{},

		Namespace:		"default",
		PullPolicy:		corev1.PullPolicy("IfNotPresent"),

		IOStreams:		genericclioptions.IOStreams{
			Out: 		os.Stdout,
			ErrOut: 	os.Stderr,
		},
	}
}

// CompleteAndVerify finishes run-time initialization of DebugOptions.
func (debugOpts *DebugOptions) CompleteAndVerify(cmd *cobra.Command, args []string) error {
	// args
	if len(args) == 0 {
		return fmt.Errorf("debugged pod or node name is required for debug")
	}

	debugOpts.TargetName = args[0]
	argsLen := cmd.ArgsLenAtDash()

	if argsLen == -1 && len(args) > 1 {
		debugOpts.Command = args[1:]
	}

	if argsLen > 0 && len(args) > argsLen {
		debugOpts.Command = args[argsLen:]
	}

	// image
	if len(debugOpts.Image) == 0 {
		imgOpts := NewImagesOptions()

		image, err := imgOpts.GetDefaultImage()
		if err != nil {
			return err
		}

		fmt.Printf("You don't specify an image, it will use the default image: %s\n" +
			"You can use `--image` to specify an image.\n", image)

		debugOpts.Image = image
	}

	if len(debugOpts.Image) > 0 && !reference.ReferenceRegexp.MatchString(debugOpts.Image) {
		return fmt.Errorf("invalid image name %q: %v", debugOpts.Image, reference.ErrReferenceInvalidFormat)
	}

	// stdin/tty
	if debugOpts.TTY || debugOpts.Interactive {
		debugOpts.In = os.Stdin
		debugOpts.Interactive = true
	}

	// env
	envStrings, err := cmd.Flags().GetStringToString("env")
	if err != nil {
		return fmt.Errorf("error getting env flag: %v", err)
	}
	for k, v := range envStrings {
		debugOpts.Env = append(debugOpts.Env, corev1.EnvVar{Name: k, Value: v})
	}

	// checklist: add check items into env
	debugOpts.Env = append(debugOpts.Env, corev1.EnvVar{
		Name:	"CHECK_LIST",
		Value:	strings.Join(debugOpts.CheckList, " "),
	})

	return nil
}

// Run generates a debug pod/node and attach to it according to command flag.
func (debugOpts *DebugOptions) Run(cmd *cobra.Command, debugFunc func (ctx context.Context) (*corev1.Pod, error)) error  {
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

	// get the kube client set
	kubeClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to create kubernetes client from file %s", adminKubeConfigPath)
	}
	debugOpts.kubeClientCorev1 = kubeClientSet.CoreV1()
	debugOpts.kubeClientRBACV1 = kubeClientSet.RbacV1()

	var (
		debugPod *corev1.Pod
		errDebug		error
	)

	// generate a debug container or pod
	debugPod, errDebug = debugFunc(ctx)
	if errDebug != nil {
		return errDebug
	}

	fmt.Println("The debug pod or container id：", debugOpts.getDebugID(debugPod))

	// will only create debug container but will not to connect it
	if len(debugOpts.Command) == 0 && !debugOpts.TTY {
		return nil
	}

	// clean the debugger container/pod
	clean := &CleanOptions{
		Namespace:			debugPod.Namespace,
		PodName: 			debugPod.Name,
		KubeClientCorev1: 	debugOpts.kubeClientCorev1,

		ContainerName: 		debugOpts.DebugContainerName,
		Config:				restConfig,
	}

	if debugOpts.Type == "node" {
		defer clean.RemovePod(ctx)
	} else {
		defer clean.ExitEphemeralContainer(ctx)
	}

	if err := debugOpts.connectPod(ctx, debugPod, restConfig); err != nil {
		return err
	}

	return nil
}

// addClusterInfoIntoEnv adds the cluster infos into DebugOptions.Env
func (debugOpts *DebugOptions) addClusterInfoIntoEnv(ctx context.Context) error {
	podsIPList, err := clusterinfo.GetPodsIP(ctx, debugOpts.kubeClientCorev1, debugOpts.Namespace)
	if err != nil {
		return err
	}
	debugOpts.Env = append(debugOpts.Env, corev1.EnvVar{
			Name: 	"POD_IP_LIST",
			Value: 	strings.Join(podsIPList, " "),
	})

	nodesIPList, err := clusterinfo.GetNodesIP(ctx, debugOpts.kubeClientCorev1)
	if err != nil {
		return err
	}
	debugOpts.Env = append(debugOpts.Env, corev1.EnvVar{
			Name: 	"NODE_IP_LIST",
			Value: 	strings.Join(nodesIPList, " "),
	})

	dnsSVCName, dnsSVCIP, dnsEndpointsIPs, err := clusterinfo.GetDNSServiceAll(ctx, debugOpts.kubeClientCorev1)
	if err != nil {
		return err
	}
	debugOpts.Env = append(debugOpts.Env,
		corev1.EnvVar{
			Name:	"KUBE_DNS_SERVICE_NAME",
			Value:	dnsSVCName,
		},
		corev1.EnvVar{
			Name:	"KUBE_DNS_SERVICE_IP",
			Value: 	dnsSVCIP,
		},
		corev1.EnvVar{
			Name:	"KUBE_DNS_ENDPOINTS_IPS",
			Value:	strings.Join(dnsEndpointsIPs, " "),
		},
	)

	return nil
}

func (debugOpts *DebugOptions) connectPod(ctx context.Context, debugPod *corev1.Pod, restConfig *rest.Config) error {
	// wait the debug container(ephemeral container) running
	debugPodRun, err := waitForContainer(ctx, debugOpts.kubeClientCorev1, debugPod.Namespace, debugPod.Name, debugOpts.DebugContainerName)
	if err != nil {
		return err
	}

	status, err := getContainerStatusByName(debugPodRun, debugOpts.DebugContainerName)
	if err != nil {
		return fmt.Errorf("error getting container status of container name %s", debugOpts.DebugContainerName)
	}

	if status.State.Terminated != nil {
		return fmt.Errorf("debug container %s terminated", debugOpts.DebugContainerName)
	}

	// begin attaching to debug container(ephemeral container)
	connectOpts := &ConnectOptions{
		NameSpace: 		debugPodRun.Namespace,
		Pod:			debugPodRun,
		Command: 		debugOpts.Command,
		ContainerName: 	debugOpts.DebugContainerName,
		Stdin: 			debugOpts.Interactive,
		TTY: 			debugOpts.TTY,
		IOStreams: 		debugOpts.IOStreams,
		Config: 		restConfig,
	}

	if err := connectOpts.Connect(); err != nil {
		return err
	}

	return nil
}

// getDebugID returns the debug ID that consists of namespace, pod name, container name
func (debugOpts *DebugOptions) getDebugID(pod *corev1.Pod) string {
	return debugOpts.DebugContainerName + DEBUG_ID_FS + pod.Name + DEBUG_ID_FS + debugOpts.Namespace
}

// setKubernetesDefaults sets default values on the provided client config for accessing the
// Kubernetes API or returns an error if any of the defaults are impossible or invalid.
func setKubernetesDefaults(config *rest.Config) error {
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

// waitForContainer watches the given pod until the container is running or terminated.
func waitForContainer(ctx context.Context, client corev1client.PodsGetter, namespace, podName, containerName string) (*corev1.Pod, error) {
	ctx, cancel := watchtools.ContextWithOptionalTimeout(ctx, 0 * time.Second)
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

		status, err := getContainerStatusByName(pod, containerName)
		if err != nil {
			return false, err
		}

		if status.State.Waiting != nil && status.State.Waiting.Reason == "ImagePullBackOff" {
			return false, fmt.Errorf("failed to pull image")
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

// getContainerStatusByName returns the container status by the containerName.
func getContainerStatusByName(pod *corev1.Pod, containerName string) (*corev1.ContainerStatus, error) {
	allContainerStatus := [][]corev1.ContainerStatus{pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses, pod.Status.EphemeralContainerStatuses}

	for _, statusSlice := range allContainerStatus {
		for _, status := range statusSlice {
			if status.Name == containerName {
				return &status, nil
			}
		}
	}

	return nil, fmt.Errorf("can not find the container %s in pod %s", containerName, pod.Name)
}

// containerNameToRef gets and returns the container names in pod.
func containerNameToRef(pod *corev1.Pod) map[string]*corev1.Container {
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