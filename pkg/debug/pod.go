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

	ephemeralContainers := pod.Spec.EphemeralContainers

	// generate an ephemeral container
	debugContainer := debugger.generateDebugContainer(pod)

	// add the ephemeral container and update the pod
	pod.Spec.EphemeralContainers = append(ephemeralContainers, *debugContainer)
	if _, err := pods.UpdateEphemeralContainers(ctx, pod.Name, pod, metav1.UpdateOptions{}); err != nil {
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
			Name:                     debugContainerName,
			Env:                      debugger.Env,
			Image:                    debugger.Image,
			ImagePullPolicy:          corev1.PullPolicy(debugger.PullPolicy),
			Stdin:                    true,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			TTY:                      true,
		},
		TargetContainerName: debugger.TargetContainer,
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
			Name:  "POD_NAME",
			Value: pod.Name,
		},
		corev1.EnvVar{
			Name:  "POD_IP",
			Value: pod.Status.PodIP,
		},
	)

	return nil
}
