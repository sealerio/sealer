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

// ConnectOptions holds the options to attach a running container.
type ConnectOptions struct {
	NameSpace		string
	ContainerName	string
	Command			[]string
	Stdin			bool
	TTY				bool
	genericclioptions.IOStreams

	Pod				*corev1.Pod
	Config			*restclient.Config
}

// Connect connects to a running container.
func (connectOpts *ConnectOptions) Connect() error {
	container, err := connectOpts.ContainerToConnect()
	if err != nil {
		return err
	}

	if connectOpts.TTY && !container.TTY {
		connectOpts.TTY =  false
	}

	// set the TTY
	t := connectOpts.SetTTY()

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
		showMotd(connectOpts.Out)
	}

	if len(connectOpts.Command) == 0 {
		if err := t.Safe(connectOpts.GetDefaultAttachFunc(container, sizeQueue)); err != nil {
			return err
		}
	} else {
		if err := t.Safe(connectOpts.GetDefaultExecFunc(container, sizeQueue)); err != nil {
			return err
		}
	}

	return nil
}

// ContainerToConnect checks if there is a container to attach, and if exists returns the container object to attach.
func (connectOpts *ConnectOptions) ContainerToConnect() (*corev1.Container, error) {
	pod := connectOpts.Pod

	if len(connectOpts.ContainerName) > 0 {
		for i := range pod.Spec.Containers {
			if pod.Spec.Containers[i].Name == connectOpts.ContainerName {
				return &pod.Spec.Containers[i], nil
			}
		}

		for i := range pod.Spec.InitContainers {
			if pod.Spec.InitContainers[i].Name == connectOpts.ContainerName {
				return &pod.Spec.InitContainers[i], nil
			}
		}

		for i := range pod.Spec.EphemeralContainers {
			if pod.Spec.EphemeralContainers[i].Name == connectOpts.ContainerName {
				return (*corev1.Container)(&pod.Spec.EphemeralContainers[i].EphemeralContainerCommon), nil
			}
		}

		return nil, fmt.Errorf("there is no container named %s", connectOpts.ContainerName)
	}

	return &pod.Spec.Containers[0], nil
}

// SetTTY handles the stdin and tty with following:
// 		1. stdin false, tty false 	--- stdout
// 		2. stdin false, tty true 	--- stdout
// 		3. stdin true, tty false 	--- stdin、stdout
// 		4. stdin true, tty true 	--- stdin、stdout、tty	--- t.Raw
// then returns a TTY object based on connectOpts.
func (connectOpts *ConnectOptions) SetTTY() TTY {
	t := TTY{
		Out: connectOpts.Out,
	}

	// Stdin is false, then tty and stdin both false
	if !connectOpts.Stdin {
		connectOpts.In = nil
		connectOpts.TTY = false
		return t
	}

	t.In = connectOpts.In
	if !connectOpts.TTY {
		return t
	}

	// check whether t.In is a terminal
	if !t.IsTerminalIn() {
		connectOpts.TTY = false
		return t
	}

	t.Raw = true

	return t
}

// GetDefaultAttachFunc returns the default attach function.
func (connectOpts *ConnectOptions) GetDefaultAttachFunc(containerToAttach *corev1.Container, sizeQueue remotecommand.TerminalSizeQueue) func() error {
	return func() error {
		restClient, err := restclient.RESTClientFor(connectOpts.Config)
		if err != nil {
			return err
		}

		req := restClient.Post().
			Resource("pods").
			Name(connectOpts.Pod.Name).
			Namespace(connectOpts.Pod.Namespace).
			SubResource("attach")
		req.VersionedParams(&corev1.PodAttachOptions{
			Container: 		containerToAttach.Name,
			Stdin: 			connectOpts.Stdin,
			Stdout: 		connectOpts.Out != nil,
			Stderr: 		connectOpts.ErrOut != nil,
			TTY:			connectOpts.TTY,
		}, scheme.ParameterCodec)

		return connectOpts.DoConnect("POST", req.URL(), sizeQueue)
	}
}

// GetDefaultExecFunc returns the default exec function.
func (connectOpts *ConnectOptions) GetDefaultExecFunc(containerToAttach *corev1.Container, sizeQueue remotecommand.TerminalSizeQueue) func() error {
	return func() error {
		restClient, err := restclient.RESTClientFor(connectOpts.Config)
		if err != nil {
			return err
		}

		req := restClient.Post().
			Resource("pods").
			Name(connectOpts.Pod.Name).
			Namespace(connectOpts.Pod.Namespace).
			SubResource("exec")
		req.VersionedParams(&corev1.PodExecOptions{
			Container: 		containerToAttach.Name,
			Command:		connectOpts.Command,
			Stdin:			connectOpts.Stdin,
			Stdout: 		connectOpts.Out != nil,
			Stderr: 		connectOpts.ErrOut != nil,
			TTY:			connectOpts.TTY,
		}, scheme.ParameterCodec)

		return connectOpts.DoConnect("POST", req.URL(), sizeQueue)
	}
}

// DoConnect executes attach to a running container with url.
func (connectOpts *ConnectOptions) DoConnect(method string, url *url.URL, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	exec, err := remotecommand.NewSPDYExecutor(connectOpts.Config, method, url)
	if err != nil {
		return err
	}

	return exec.Stream(remotecommand.StreamOptions{
		Stdin: 					connectOpts.In,
		Stdout: 				connectOpts.Out,
		Stderr: 				connectOpts.ErrOut,
		Tty: 					connectOpts.TTY,
		TerminalSizeQueue:  	terminalSizeQueue,
	})
}