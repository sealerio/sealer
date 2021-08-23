package debug

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

type DebugPodOptions struct {
	*DebugOptions

	TargetContainer		string		// target container to share the namespace
}

const POD_DEBUG_PREFIX = "pod-debugger"

var debugPodOptions *DebugPodOptions

func NewDebugPodOptions(debugOpts *DebugOptions) *DebugPodOptions{
	return &DebugPodOptions{
		DebugOptions: debugOpts,
	}
}

func NewDebugPod(options *DebugOptions) *cobra.Command {
	debugPodOptions := NewDebugPodOptions(options)

	cmd := &cobra.Command{
		Use:     "pod",
		Short:   "Debug pod or container",
		Long:    "",
		Example: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			options.Type = "pod"

			if err := debugPodOptions.CompleteAndVerify(cmd, args); err != nil {
				return err
			}
			if err := debugPodOptions.Run(cmd, debugPodOptions.DebugPod); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&debugPodOptions.TargetContainer, "container", "c", "", "The container to be debugged.")

	return cmd
}

func (debugOpts *DebugPodOptions) DebugPod(ctx context.Context) (*corev1.Pod, error) {
	// get the target pod object
	targetPod, err := debugOpts.kubeClientCorev1.Pods(debugOpts.Namespace).Get(ctx, debugOpts.TargetName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the target pod %s", debugOpts.TargetName)
	}

	if err := debugOpts.addPodInfoIntoEnv(targetPod); err != nil {
		return nil, err
	}
	if err := debugOpts.addClusterInfoIntoEnv(ctx); err != nil {
		return nil, err
	}

	// add an ephemeral container into target pod and used as a debug container
	debugPod, err := debugOpts.debugPodByEphemeralContainer(ctx, targetPod)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add an ephemeral container into pod: %s", targetPod.Name)
	}

	return debugPod, nil
}

// debugPodByEphemeralContainer runs an ephemeral container in target pod and use as a debug container.
func (debugOpts *DebugPodOptions) debugPodByEphemeralContainer(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	// get ephemeral containers
	pods := debugOpts.kubeClientCorev1.Pods(pod.Namespace)
	ec, err := pods.GetEphemeralContainers(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		if serr, ok := err.(*apierrors.StatusError); ok && serr.Status().Reason == metav1.StatusReasonNotFound && serr.ErrStatus.Details.Name == "" {
			return nil, errors.Wrapf(err, "ephemeral container are disabled for this cluster")
		}
		return nil, err
	}

	// generate an ephemeral container
	debugContainer := debugOpts.generateDebugContainer(pod)

	// add the ephemeral container and update the pod
	ec.EphemeralContainers = append(ec.EphemeralContainers, *debugContainer)
	_, err = pods.UpdateEphemeralContainers(ctx, pod.Name, ec, metav1.UpdateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "error updating ephermeral containers")
	}

	return pod, nil
}

// generateDebugContainer returns an ephemeral container suitable for use as a debug container in the given pod.
func (debugOpts *DebugPodOptions) generateDebugContainer(pod *corev1.Pod) *corev1.EphemeralContainer {
	debugContainerName := debugOpts.getDebugContainerName(pod)
	debugOpts.DebugContainerName = debugContainerName

	if len(debugOpts.TargetContainer) == 0 {
		debugOpts.TargetContainer = pod.Spec.Containers[0].Name
	}

	ec := &corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name: 						debugContainerName,
			Env:						debugOpts.Env,
			Image:						debugOpts.Image,
			ImagePullPolicy: 			debugOpts.PullPolicy,
			Stdin:						true,
			TerminationMessagePolicy: 	corev1.TerminationMessageReadFile,
			TTY: 						true,
		},
		TargetContainerName: 			debugOpts.TargetContainer,
	}

	return ec
}

// getDebugContainerName generates and returns the debug container name.
func (debugOpts *DebugPodOptions) getDebugContainerName(pod *corev1.Pod) string {
	if len(debugOpts.DebugContainerName) > 0 {
		return debugOpts.DebugContainerName
	}

	name := debugOpts.DebugContainerName
	containerByName := containerNameToRef(pod)
	for len(name) == 0 || containerByName[name] != nil {
		name = fmt.Sprintf("%s-%s", POD_DEBUG_PREFIX, utilrand.String(5))
	}

	return name
}

// addPodInfoIntoEnv adds pod info into env
func (debugOpts *DebugPodOptions) addPodInfoIntoEnv(pod *corev1.Pod) error {
	if pod == nil {
		return fmt.Errorf("pod must not nil")
	}

	debugOpts.Env = append(debugOpts.Env,
		corev1.EnvVar{
			Name: "POD_NAME",
			Value: pod.Name,
		},
		corev1.EnvVar{
			Name: "POD_IP",
			Value: pod.Status.PodIP,
		},
	)

	return nil
}