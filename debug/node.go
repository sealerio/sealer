package debug

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

type DebugNodeOptions struct {
	*DebugOptions
}

const NODE_DEBUG_PREFIX = "node-debugger"

var debugNodeOptions *DebugNodeOptions

func NewDebugNodeOptions(debutOpt *DebugOptions) *DebugNodeOptions{
	return &DebugNodeOptions{
		DebugOptions:	debutOpt,
	}
}

func NewDebugNode(options *DebugOptions) *cobra.Command {
	debugNodeOptions := NewDebugNodeOptions(options)

	cmd := &cobra.Command{
		Use:     "node",
		Short:   "Debug node",
		Long:    "",
		Example: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			options.Type = "node"

			if err := debugNodeOptions.CompleteAndVerify(cmd, args); err != nil {
				return err
			}
			if err := debugNodeOptions.Run(cmd, debugNodeOptions.DebugNode); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func (debugOpts *DebugNodeOptions) DebugNode(ctx context.Context) (*corev1.Pod, error) {
	// get the target node object
	targetNode, err := debugOpts.kubeClientCorev1.Nodes().Get(ctx, debugOpts.TargetName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find the target node %s", debugOpts.TargetName)
	}

	if err := debugOpts.addClusterInfoIntoEnv(ctx); err != nil {
		return nil, err
	}

	// add a pod into target node
	debugPod, err := debugOpts.debugNodeByPod(ctx, targetNode)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add a pod into node: %s", targetNode.Name)
	}

	return debugPod, nil
}

// debugNodeByPod runs a pod in target node and use as a debug pod.
func (debugOpts *DebugNodeOptions) debugNodeByPod(ctx context.Context, node *corev1.Node) (*corev1.Pod, error) {
	pods := debugOpts.kubeClientCorev1.Pods(debugOpts.Namespace)

	debugPod, err := pods.Create(ctx, debugOpts.generateDebugPod(node.Name), metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return debugPod, nil
}

// generateDebugPod generates a debug pod that schedules on the specified node.
// The generated pod will run in the host PID, Network & IPC namespace, and it will
// have the node's filesystem mounted at /hostfs.
func (debugOpts *DebugNodeOptions) generateDebugPod(nodeName string) *corev1.Pod {
	cn := "debugger"
	if len(debugOpts.DebugContainerName) > 0 {
		cn = debugOpts.DebugContainerName
	} else {
		debugOpts.DebugContainerName = cn
	}

	pn := fmt.Sprintf("%s-%s-%s", NODE_DEBUG_PREFIX, nodeName, utilrand.String(7))

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: pn,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: 						cn,
					Env: 						debugOpts.Env,
					Image: 						debugOpts.Image,
					ImagePullPolicy: 			debugOpts.PullPolicy,
					Stdin: 						true,
					TTY:						true,
					TerminationMessagePolicy: 	corev1.TerminationMessageReadFile,
					VolumeMounts: []corev1.VolumeMount{
						{
							MountPath: 	"/hostfs",
							Name:		"host-root",
						},
					},
				},
			},
			HostIPC: 			true,
			HostNetwork: 		true,
			HostPID: 			true,
			NodeName: 			nodeName,
			RestartPolicy: 		corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{
				{
					Name:	"host-root",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{Path:"/"},
					},
				},
			},
		},
	}

	return pod
}