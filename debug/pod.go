package debug

import (
	"context"
	"fmt"

	"github.com/alibaba/sealer/common"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

func NewDebugPodCommand(options *DebugOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use: 	 "pod",
		Short:   "Debug pod or container",
		Args:	 cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			debugger := NewDebugger(options)
			debugger.AdminKubeConfigPath = common.KubeAdminConf
			debugger.Type = TypeDebugPod
			debugger.Motd = SEALER_DEBUG_MOTD

			imager := NewDebugImagesManager()

			if err := debugger.CompleteAndVerifyOptions(cmd, args, imager); err != nil {
				return err
			}
			str, err := debugger.Run()
			if err != nil {
				return err
			}
			fmt.Println("The debug ID:", str)

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.TargetContainer, "container", "c", "", "The container to be debugged.")

	return cmd
}

func (debugger *Debugger) DebugPod(ctx context.Context) (*corev1.Pod, error) {
	// get the target pod object
	targetPod, err := debugger.kubeClientCorev1.Pods(debugger.Namespace).Get(ctx, debugger.TargetName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the target pod %s", debugger.TargetName)
	}

	if err := debugger.addPodInfoIntoEnv(targetPod); err != nil {
		return nil, err
	}
	if err := debugger.addClusterInfoIntoEnv(ctx); err != nil {
		return nil, err
	}

	// add an ephemeral container into target pod and used as a debug container
	debugPod, err := debugger.debugPodByEphemeralContainer(ctx, targetPod)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add an ephemeral container into pod: %s", targetPod.Name)
	}

	return debugPod, nil
}

// debugPodByEphemeralContainer runs an ephemeral container in target pod and use as a debug container.
func (debugger *Debugger) debugPodByEphemeralContainer(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	// get ephemeral containers
	pods := debugger.kubeClientCorev1.Pods(pod.Namespace)
	ec, err := pods.GetEphemeralContainers(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		if serr, ok := err.(*apierrors.StatusError); ok && serr.Status().Reason == metav1.StatusReasonNotFound && serr.ErrStatus.Details.Name == "" {
			return nil, errors.Wrapf(err, "ephemeral container are disabled for this cluster")
		}
		return nil, err
	}

	// generate an ephemeral container
	debugContainer := debugger.generateDebugContainer(pod)

	// add the ephemeral container and update the pod
	ec.EphemeralContainers = append(ec.EphemeralContainers, *debugContainer)
	_, err = pods.UpdateEphemeralContainers(ctx, pod.Name, ec, metav1.UpdateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "error updating ephermeral containers")
	}

	return pod, nil
}

// generateDebugContainer returns an ephemeral container suitable for use as a debug container in the given pod.
func (debugger *Debugger) generateDebugContainer(pod *corev1.Pod) *corev1.EphemeralContainer {
	debugContainerName := debugger.getDebugContainerName(pod)
	debugger.DebugContainerName = debugContainerName

	if len(debugger.TargetContainer) == 0 {
		debugger.TargetContainer = pod.Spec.Containers[0].Name
	}

	ec := &corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name: 						debugContainerName,
			Env:						debugger.Env,
			Image:						debugger.Image,
			ImagePullPolicy: 			corev1.PullPolicy(debugger.PullPolicy),
			Stdin:						true,
			TerminationMessagePolicy: 	corev1.TerminationMessageReadFile,
			TTY: 						true,
		},
		TargetContainerName: 			debugger.TargetContainer,
	}

	return ec
}

// getDebugContainerName generates and returns the debug container name.
func (debugger *Debugger) getDebugContainerName(pod *corev1.Pod) string {
	if len(debugger.DebugContainerName) > 0 {
		return debugger.DebugContainerName
	}

	name := debugger.DebugContainerName
	containerByName := ContainerNameToRef(pod)
	for len(name) == 0 || containerByName[name] != nil {
		name = fmt.Sprintf("%s-%s", PodDebugPrefix, utilrand.String(5))
	}

	return name
}

// addPodInfoIntoEnv adds pod info into env
func (debugger *Debugger) addPodInfoIntoEnv(pod *corev1.Pod) error {
	if pod == nil {
		return fmt.Errorf("pod must not nil")
	}

	debugger.Env = append(debugger.Env,
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