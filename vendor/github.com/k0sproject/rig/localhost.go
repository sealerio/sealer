package rig

import (
	"bufio"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/alessio/shellescape"
	"github.com/k0sproject/rig/exec"
	ps "github.com/k0sproject/rig/powershell"
	"github.com/kballard/go-shellquote"
)

const name = "[local] localhost"

// Localhost is a direct localhost connection
type Localhost struct {
	Enabled bool `yaml:"enabled" validate:"required,eq=true" default:"true"`
}

// Protocol returns the protocol name, "Local"
func (c *Localhost) Protocol() string {
	return "Local"
}

// IPAddress returns the connection address
func (c *Localhost) IPAddress() string {
	return "127.0.0.1"
}

// String returns the connection's printable name
func (c *Localhost) String() string {
	return name
}

// IsConnected for local connections is always true
func (c *Localhost) IsConnected() bool {
	return true
}

// IsWindows is true when running on a windows host
func (c *Localhost) IsWindows() bool {
	return runtime.GOOS == "windows"
}

// Connect on local connection does nothing
func (c *Localhost) Connect() error {
	return nil
}

// Disconnect on local connection does nothing
func (c *Localhost) Disconnect() {}

// Exec executes a command on the host
func (c *Localhost) Exec(cmd string, opts ...exec.Option) error {
	o := exec.Build(opts...)
	command, err := c.command(cmd, o)
	if err != nil {
		return err
	}

	if o.Stdin != "" {
		o.LogStdin(name)

		command.Stdin = strings.NewReader(o.Stdin)
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return err
	}

	o.LogCmd(name, cmd)

	if err := command.Start(); err != nil {
		return err
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		if o.Writer == nil {
			outputScanner := bufio.NewScanner(stdout)

			for outputScanner.Scan() {
				o.AddOutput(name, outputScanner.Text()+"\n", "")
			}
		} else {
			if _, err := io.Copy(o.Writer, stdout); err != nil {
				o.LogErrorf("%s: failed to stream stdout", c, err.Error())
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()

		outputScanner := bufio.NewScanner(stderr)

		for outputScanner.Scan() {
			o.AddOutput(name, "", outputScanner.Text()+"\n")
		}
	}()

	err = command.Wait()
	wg.Wait()
	return err
}

func (c *Localhost) command(cmd string, o *exec.Options) (*osexec.Cmd, error) {
	cmd, err := o.Command(cmd)
	if err != nil {
		return nil, err
	}

	if c.IsWindows() {
		return osexec.Command(cmd), nil
	}

	return osexec.Command("bash", "-c", "--", cmd), nil
}

// Upload copies a larger file to another path on the host.
func (c *Localhost) Upload(src, dst string, opts ...exec.Option) error {
	var remoteErr error
	defer func() {
		if remoteErr != nil {
			if c.IsWindows() {
				_ = c.Exec(fmt.Sprintf(`del %s`, ps.DoubleQuote(dst)))
			} else {
				_ = c.Exec(fmt.Sprintf(`rm -f -- %s`, shellescape.Quote(dst)))
			}
		}
	}()

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, remoteErr = io.Copy(out, in)
	return remoteErr
}

// ExecInteractive executes a command on the host and copies stdin/stdout/stderr from local host
func (c *Localhost) ExecInteractive(cmd string) error {
	if cmd == "" {
		cmd = os.Getenv("SHELL") + " -l"
	}

	if cmd == " -l" {
		cmd = "cmd"
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	pa := os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Dir:   cwd,
	}

	parts, err := shellquote.Split(cmd)
	if err != nil {
		return err
	}

	proc, err := os.StartProcess(parts[0], parts[1:], &pa)
	if err != nil {
		return err
	}

	_, err = proc.Wait()
	println("shell exited")
	return err
}
