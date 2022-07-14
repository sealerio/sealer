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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

// DebugNode can debug a node.
func (debugger *Debugger) DebugNode(ctx context.Context) (*corev1.Pod, error) {
	// get the target node object
	targetNode, err := debugger.kubeClientCorev1.Nodes().Get(ctx, debugger.TargetName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find the target node %s", debugger.TargetName)
	}

	if err := debugger.addClusterInfoIntoEnv(ctx); err != nil {
		return nil, err
	}

	// add a pod into target node
	debugPod, err := debugger.debugNodeByPod(ctx, targetNode)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add a pod into node: %s", targetNode.Name)
	}

	return debugPod, nil
}

// debugNodeByPod runs a pod in target node and use as a debug pod.
func (debugger *Debugger) debugNodeByPod(ctx context.Context, node *corev1.Node) (*corev1.Pod, error) {
	pods := debugger.kubeClientCorev1.Pods(debugger.Namespace)

	debugPod, err := pods.Create(ctx, debugger.generateDebugPod(node.Name), metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return debugPod, nil
}

// generateDebugPod generates a debug pod that schedules on the specified node.
// The generated pod will run in the host PID, Network & IPC namespace, and it will
// have the node's filesystem mounted at /hostfs.
func (debugger *Debugger) generateDebugPod(nodeName string) *corev1.Pod {
	cn := PodDebugPrefix
	if len(debugger.DebugContainerName) > 0 {
		cn = debugger.DebugContainerName
	} else {
		debugger.DebugContainerName = cn
	}

	pn := fmt.Sprintf("%s-%s-%s", NodeDebugPrefix, nodeName, utilrand.String(7))

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: pn,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:                     cn,
					Env:                      debugger.Env,
					Image:                    debugger.Image,
					ImagePullPolicy:          corev1.PullPolicy(debugger.PullPolicy),
					Stdin:                    true,
					TTY:                      true,
					TerminationMessagePolicy: corev1.TerminationMessageReadFile,
					VolumeMounts: []corev1.VolumeMount{
						{
							MountPath: "/hostfs",
							Name:      "host-root",
						},
					},
				},
			},
			HostIPC:       true,
			HostNetwork:   true,
			HostPID:       true,
			NodeName:      nodeName,
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{
				{
					Name: "host-root",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{Path: "/"},
					},
				},
			},
		},
	}

	return pod
}
