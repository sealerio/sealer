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
	"fmt"
	"net/url"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// Connector holds the options to connect a running container.
type Connector struct {
	NameSpace     string
	ContainerName string
	Command       []string
	Stdin         bool
	TTY           bool
	genericclioptions.IOStreams

	Motd string

	Pod    *corev1.Pod
	Config *restclient.Config
}

// Connect connects to a running container.
func (connector *Connector) Connect() error {
	container, err := connector.ContainerToConnect()
	if err != nil {
		return err
	}

	if connector.TTY && !container.TTY {
		connector.TTY = false
	}

	// set the TTY
	t := connector.SetTTY()

	// get the terminal size queue
	var sizeQueue remotecommand.TerminalSizeQueue
	if t.Raw {
		// this spawns a goroutine to monitor/update the terminal size
		if size := t.GetSize(); size != nil {
			sizePlusOne := *size
			sizePlusOne.Width++
			sizePlusOne.Height++

			sizeQueue = t.MonitorSize(&sizePlusOne, size)
		}

		showMotd(connector.Out, connector.Motd)
	}

	if len(connector.Command) == 0 {
		if err := t.Safe(connector.GetDefaultAttachFunc(container, sizeQueue)); err != nil {
			return err
		}
	} else {
		if err := t.Safe(connector.GetDefaultExecFunc(container, sizeQueue)); err != nil {
			return err
		}
	}

	return nil
}

// ContainerToConnect checks if there is a container to attach, and if exists returns the container object to attach.
func (connector *Connector) ContainerToConnect() (*corev1.Container, error) {
	pod := connector.Pod

	if len(connector.ContainerName) > 0 {
		for i := range pod.Spec.Containers {
			if pod.Spec.Containers[i].Name == connector.ContainerName {
				return &pod.Spec.Containers[i], nil
			}
		}

		for i := range pod.Spec.InitContainers {
			if pod.Spec.InitContainers[i].Name == connector.ContainerName {
				return &pod.Spec.InitContainers[i], nil
			}
		}

		for i := range pod.Spec.EphemeralContainers {
			if pod.Spec.EphemeralContainers[i].Name == connector.ContainerName {
				return (*corev1.Container)(&pod.Spec.EphemeralContainers[i].EphemeralContainerCommon), nil
			}
		}

		return nil, fmt.Errorf("there is no container named %s", connector.ContainerName)
	}

	return &pod.Spec.Containers[0], nil
}

// SetTTY handles the stdin and tty with following:
//  1. stdin false, tty false 	--- stdout
//  2. stdin false, tty true 	--- stdout
//  3. stdin true, tty false 	--- stdin、stdout
//  4. stdin true, tty true 	--- stdin、stdout、tty	--- t.Raw
//
// then returns a TTY object based on connectOpts.

func (connector *Connector) SetTTY() TTY {
	t := TTY{
		Out: connector.Out,
	}

	// Stdin is false, then tty and stdin both false
	if !connector.Stdin {
		connector.In = nil
		connector.TTY = false
		return t
	}

	t.In = connector.In
	if !connector.TTY {
		return t
	}

	// check whether t.In is a terminal
	if !t.IsTerminalIn() {
		connector.TTY = false
		return t
	}

	t.Raw = true

	return t
}

// GetDefaultAttachFunc returns the default attach function.
func (connector *Connector) GetDefaultAttachFunc(containerToAttach *corev1.Container, sizeQueue remotecommand.TerminalSizeQueue) func() error {
	return func() error {
		restClient, err := restclient.RESTClientFor(connector.Config)
		if err != nil {
			return err
		}

		req := restClient.Post().
			Resource("pods").
			Name(connector.Pod.Name).
			Namespace(connector.Pod.Namespace).
			SubResource("attach")
		req.VersionedParams(&corev1.PodAttachOptions{
			Container: containerToAttach.Name,
			Stdin:     connector.Stdin,
			Stdout:    connector.Out != nil,
			Stderr:    connector.ErrOut != nil,
			TTY:       connector.TTY,
		}, scheme.ParameterCodec)

		return connector.DoConnect("POST", req.URL(), sizeQueue)
	}
}

// GetDefaultExecFunc returns the default exec function.
func (connector *Connector) GetDefaultExecFunc(containerToAttach *corev1.Container, sizeQueue remotecommand.TerminalSizeQueue) func() error {
	return func() error {
		restClient, err := restclient.RESTClientFor(connector.Config)
		if err != nil {
			return err
		}

		req := restClient.Post().
			Resource("pods").
			Name(connector.Pod.Name).
			Namespace(connector.Pod.Namespace).
			SubResource("exec")
		req.VersionedParams(&corev1.PodExecOptions{
			Container: containerToAttach.Name,
			Command:   connector.Command,
			Stdin:     connector.Stdin,
			Stdout:    connector.Out != nil,
			Stderr:    connector.ErrOut != nil,
			TTY:       connector.TTY,
		}, scheme.ParameterCodec)

		return connector.DoConnect("POST", req.URL(), sizeQueue)
	}
}

// DoConnect executes attach to a running container with url.
func (connector *Connector) DoConnect(method string, url *url.URL, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	exec, err := remotecommand.NewSPDYExecutor(connector.Config, method, url)
	if err != nil {
		return err
	}

	return exec.Stream(remotecommand.StreamOptions{
		Stdin:             connector.In,
		Stdout:            connector.Out,
		Stderr:            connector.ErrOut,
		Tty:               connector.TTY,
		TerminalSizeQueue: terminalSizeQueue,
	})
}
