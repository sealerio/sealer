package rig

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/k0sproject/rig/exec"
	"github.com/k0sproject/rig/log"
	ps "github.com/k0sproject/rig/powershell"
	"github.com/mitchellh/go-homedir"

	"github.com/masterzen/winrm"
)

// WinRM describes a WinRM connection with its configuration options
type WinRM struct {
	Address       string `yaml:"address" validate:"required,hostname|ip"`
	User          string `yaml:"user" validate:"omitempty,gt=2" default:"Administrator"`
	Port          int    `yaml:"port" default:"5985" validate:"gt=0,lte=65535"`
	Password      string `yaml:"password,omitempty"`
	UseHTTPS      bool   `yaml:"useHTTPS" default:"false"`
	Insecure      bool   `yaml:"insecure" default:"false"`
	UseNTLM       bool   `yaml:"useNTLM" default:"false"`
	CACertPath    string `yaml:"caCertPath,omitempty" validate:"omitempty,file"`
	CertPath      string `yaml:"certPath,omitempty" validate:"omitempty,file"`
	KeyPath       string `yaml:"keyPath,omitempty" validate:"omitempty,file"`
	TLSServerName string `yaml:"tlsServerName,omitempty" validate:"omitempty,hostname|ip"`
	Bastion       *SSH   `yaml:"bastion,omitempty"`

	name string

	caCert []byte
	key    []byte
	cert   []byte

	client *winrm.Client
}

// SetDefaults sets various default values
func (c *WinRM) SetDefaults() {
	if p, err := homedir.Expand(c.CACertPath); err == nil {
		c.CACertPath = p
	}

	if p, err := homedir.Expand(c.CertPath); err == nil {
		c.CertPath = p
	}

	if p, err := homedir.Expand(c.KeyPath); err == nil {
		c.KeyPath = p
	}

	if c.Port == 5985 && c.UseHTTPS {
		c.Port = 5986
	}
}

// Protocol returns the protocol name, "WinRM"
func (c *WinRM) Protocol() string {
	return "WinRM"
}

// IPAddress returns the connection address
func (c *WinRM) IPAddress() string {
	return c.Address
}

// String returns the connection's printable name
func (c *WinRM) String() string {
	if c.name == "" {
		c.name = fmt.Sprintf("[winrm] %s:%d", c.Address, c.Port)
	}

	return c.name
}

// IsConnected returns true if the client is connected
func (c *WinRM) IsConnected() bool {
	return c.client != nil
}

// IsWindows always returns true on winrm
func (c *WinRM) IsWindows() bool {
	return true
}

func (c *WinRM) loadCertificates() error {
	c.caCert = nil
	if c.CACertPath != "" {
		ca, err := os.ReadFile(c.CACertPath)
		if err != nil {
			return err
		}
		c.caCert = ca
	}

	c.cert = nil
	if c.CertPath != "" {
		cert, err := os.ReadFile(c.CertPath)
		if err != nil {
			return err
		}
		c.cert = cert
	}

	c.key = nil
	if c.KeyPath != "" {
		key, err := os.ReadFile(c.KeyPath)
		if err != nil {
			return err
		}
		c.key = key
	}

	return nil
}

// Connect opens the WinRM connection
func (c *WinRM) Connect() error {
	if err := c.loadCertificates(); err != nil {
		return fmt.Errorf("%s: failed to load certificates: %s", c, err)
	}

	endpoint := &winrm.Endpoint{
		Host:          c.Address,
		Port:          c.Port,
		HTTPS:         c.UseHTTPS,
		Insecure:      c.Insecure,
		TLSServerName: c.TLSServerName,
		Timeout:       60 * time.Second,
	}

	if len(c.caCert) > 0 {
		endpoint.CACert = c.caCert
	}

	if len(c.cert) > 0 {
		endpoint.Cert = c.cert
	}

	if len(c.key) > 0 {
		endpoint.Key = c.key
	}

	params := winrm.DefaultParameters

	if c.Bastion != nil {
		err := c.Bastion.Connect()
		if err != nil {
			return err
		}
		params.Dial = c.Bastion.client.Dial
	}

	if c.UseNTLM {
		params.TransportDecorator = func() winrm.Transporter { return &winrm.ClientNTLM{} }
	}

	if c.UseHTTPS && len(c.cert) > 0 {
		params.TransportDecorator = func() winrm.Transporter { return &winrm.ClientAuthRequest{} }
	}

	client, err := winrm.NewClientWithParameters(endpoint, c.User, c.Password, params)

	if err != nil {
		return err
	}

	log.Debugf("%s: testing connection", c)
	_, err = client.Run("echo ok", io.Discard, io.Discard)
	if err != nil {
		return err
	}
	log.Debugf("%s: test passed", c)

	c.client = client

	return nil
}

// Disconnect closes the WinRM connection
func (c *WinRM) Disconnect() {
	c.client = nil
}

// Exec executes a command on the host
func (c *WinRM) Exec(cmd string, opts ...exec.Option) error {
	o := exec.Build(opts...)
	shell, err := c.client.CreateShell()
	if err != nil {
		return err
	}
	defer shell.Close()

	o.LogCmd(c.String(), cmd)

	command, err := shell.Execute(cmd)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	if o.Stdin != "" {
		o.LogStdin(c.String())
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer command.Stdin.Close()
			_, _ = command.Stdin.Write([]byte(o.Stdin))
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if o.Writer == nil {
			outputScanner := bufio.NewScanner(command.Stdout)

			for outputScanner.Scan() {
				o.AddOutput(c.String(), outputScanner.Text()+"\n", "")
			}

			if err := outputScanner.Err(); err != nil {
				o.LogErrorf("%s: %s", c, err.Error())
			}
			command.Stdout.Close()
		} else {
			if _, err := io.Copy(o.Writer, command.Stdout); err != nil {
				o.LogErrorf("%s: failed to stream stdout", c, err.Error())
			}
		}
	}()

	gotErrors := false

	wg.Add(1)
	go func() {
		defer wg.Done()
		outputScanner := bufio.NewScanner(command.Stderr)

		for outputScanner.Scan() {
			gotErrors = true
			o.AddOutput(c.String(), "", outputScanner.Text()+"\n")
		}

		if err := outputScanner.Err(); err != nil {
			gotErrors = true
			o.LogErrorf("%s: %s", c, err.Error())
		}
		command.Stderr.Close()
	}()

	command.Wait()

	wg.Wait()

	command.Close()

	if command.ExitCode() > 0 || (!o.AllowWinStderr && gotErrors) {
		return fmt.Errorf("command failed (received output to stderr on windows)")
	}

	return nil
}

// ExecInteractive executes a command on the host and copies stdin/stdout/stderr from local host
func (c *WinRM) ExecInteractive(cmd string) error {
	if cmd == "" {
		cmd = "cmd"
	}
	_, err := c.client.RunWithInput(cmd, os.Stdout, os.Stderr, os.Stdin)
	return err
}

// Upload uploads a file from local src path to remote path dst
func (c *WinRM) Upload(src, dst string, opts ...exec.Option) error {
	var err error
	defer func() {
		if err != nil {
			_ = c.Exec(fmt.Sprintf(`del %s`, ps.DoubleQuote(dst)), opts...)
		}
	}()
	psCmd := ps.UploadCmd(dst)
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}
	sha256DigestLocalObj := sha256.New()
	sha256DigestLocal := ""
	sha256DigestRemote := ""
	srcSize := uint64(stat.Size())
	var bytesSent uint64
	var realSent uint64
	var fdClosed bool
	fd, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if !fdClosed {
			_ = fd.Close()
			fdClosed = true
		}
	}()
	shell, err := c.client.CreateShell()
	if err != nil {
		return err
	}
	defer shell.Close()
	o := exec.Build(opts...)
	upcmd, err := o.Command("powershell -ExecutionPolicy Unrestricted -EncodedCommand " + psCmd)
	if err != nil {
		return err
	}

	cmd, err := shell.Execute(upcmd)
	if err != nil {
		return err
	}

	// Create a dummy request to get its length
	dummy := winrm.NewSendInputRequest("dummydummydummy", "dummydummydummy", "dummydummydummy", []byte(""), false, winrm.DefaultParameters)
	maxInput := len(dummy.String()) - 100
	bufferCapacity := (winrm.DefaultParameters.EnvelopeSize - maxInput) / 4 * 3
	base64LineBufferCapacity := bufferCapacity/3*4 + 2
	base64LineBuffer := make([]byte, base64LineBufferCapacity)
	base64LineBuffer[base64LineBufferCapacity-2] = '\r'
	base64LineBuffer[base64LineBufferCapacity-1] = '\n'
	buffer := make([]byte, bufferCapacity)
	var bufferLength int

	var ended bool

	for {
		var n int
		n, err = fd.Read(buffer)
		bufferLength += n
		if err != nil {
			break
		}
		if bufferLength == bufferCapacity {
			base64.StdEncoding.Encode(base64LineBuffer, buffer)
			bytesSent += uint64(bufferLength)
			_, _ = sha256DigestLocalObj.Write(buffer)
			if bytesSent >= srcSize {
				ended = true
				sha256DigestLocal = hex.EncodeToString(sha256DigestLocalObj.Sum(nil))
			}
			b, err := cmd.Stdin.Write(base64LineBuffer)
			realSent += uint64(b)
			if ended {
				cmd.Stdin.Close()
			}

			bufferLength = 0
			if err != nil {
				return err
			}
		}
	}
	_ = fd.Close()
	fdClosed = true
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		cmd.Close()
		return err
	}
	if !ended {
		_, _ = sha256DigestLocalObj.Write(buffer[:bufferLength])
		sha256DigestLocal = hex.EncodeToString(sha256DigestLocalObj.Sum(nil))
		base64.StdEncoding.Encode(base64LineBuffer, buffer[:bufferLength])
		i := base64.StdEncoding.EncodedLen(bufferLength)
		base64LineBuffer[i] = '\r'
		base64LineBuffer[i+1] = '\n'
		_, err = cmd.Stdin.Write(base64LineBuffer[:i+2])
		if err != nil {
			if !strings.Contains(err.Error(), ps.PipeHasEnded) && !strings.Contains(err.Error(), ps.PipeIsBeingClosed) {
				cmd.Close()
				return err
			}
			// ignore pipe errors that results from passing true to cmd.SendInput
		}
		cmd.Stdin.Close()
	}
	var wg sync.WaitGroup
	wg.Add(2)
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	go func() {
		defer wg.Done()
		_, err = io.Copy(&stderr, cmd.Stderr)
		if err != nil {
			stderr.Reset()
		}
	}()
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(cmd.Stdout)
		for scanner.Scan() {
			var output struct {
				Sha256 string `json:"sha256"`
			}
			if json.Unmarshal(scanner.Bytes(), &output) == nil {
				sha256DigestRemote = output.Sha256
			} else {
				_, _ = stdout.Write(scanner.Bytes())
				_, _ = stdout.WriteString("\n")
			}
		}
		if err := scanner.Err(); err != nil {
			stdout.Reset()
		}
	}()
	cmd.Wait()
	wg.Wait()

	if cmd.ExitCode() != 0 {
		return fmt.Errorf("non-zero exit code: %d during upload", cmd.ExitCode())
	}
	if sha256DigestRemote == "" {
		return fmt.Errorf("copy file command did not output the expected JSON to stdout but exited with code 0")
	} else if sha256DigestRemote != sha256DigestLocal {
		return fmt.Errorf("copy file checksum mismatch (local = %s, remote = %s)", sha256DigestLocal, sha256DigestRemote)
	}

	return nil
}
