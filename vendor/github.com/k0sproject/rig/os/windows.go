package os

import (
	"bufio"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"time"

	"github.com/k0sproject/rig/exec"
	"github.com/k0sproject/rig/log"
	ps "github.com/k0sproject/rig/powershell"
)

// Windows is the base package for windows OS support
type Windows struct{}

// Kind returns "windows"
func (c Windows) Kind() string {
	return "windows"
}

const privCheck = `"$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent()); if (!$currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { $host.SetShouldExit(1) }"`

// CheckPrivilege returns an error if the user does not have admin access to the host
func (c Windows) CheckPrivilege(h Host) error {
	if h.Exec(ps.Cmd(privCheck)) != nil {
		return fmt.Errorf("user does not have administrator rights on the host")
	}

	return nil
}

// InstallPackage enables an optional windows feature
func (c Windows) InstallPackage(h Host, s ...string) error {
	for _, n := range s {
		err := h.Exec(ps.Cmd(fmt.Sprintf("Enable-WindowsOptionalFeature -Online -FeatureName %s -All", n)))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c Windows) InstallFile(h Host, src, dst, _ string) error {
	return h.Execf("move /y %s %s", ps.DoubleQuote(src), ps.DoubleQuote(dst), exec.Sudo(h))
}

// Pwd returns the current working directory
func (c Windows) Pwd(h Host) string {
	if pwd, err := h.ExecOutput("echo %cd%"); err == nil {
		return pwd
	}

	return ""
}

// JoinPath joins a path
func (c Windows) JoinPath(parts ...string) string {
	return strings.Join(parts, "\\")
}

// Hostname resolves the short hostname
func (c Windows) Hostname(h Host) string {
	output, err := h.ExecOutput("powershell $env:COMPUTERNAME")
	if err != nil {
		return "localhost"
	}

	return strings.TrimSpace(output)
}

// LongHostname resolves the FQDN (long) hostname
func (c Windows) LongHostname(h Host) string {
	output, err := h.ExecOutput("powershell ([System.Net.Dns]::GetHostByName(($env:COMPUTERNAME))).Hostname")
	if err != nil {
		return "localhost.localdomain"
	}
	return strings.TrimSpace(output)
}

// IsContainer returns true if the host is actually a container (always false on windows for now)
func (c Windows) IsContainer(_ Host) bool {
	return false
}

// FixContainer makes a container work like a real host (does nothing on windows for now)
func (c Windows) FixContainer(_ Host) error {
	return nil
}

// SELinuxEnabled is true when SELinux is enabled (always false on windows for now)
func (c Windows) SELinuxEnabled(_ Host) bool {
	return false
}

// WriteFile writes file to host with given contents. Do not use for large files.
// The permissions argument is ignored on windows.
func (c Windows) WriteFile(h Host, path string, data string, permissions string) error {
	if data == "" {
		return fmt.Errorf("empty content in WriteFile to %s", path)
	}

	if path == "" {
		return fmt.Errorf("empty path in WriteFile")
	}

	tempFile, err := h.ExecOutput("powershell -Command \"New-TemporaryFile | Write-Host\"")
	if err != nil {
		return err
	}
	defer c.deleteTempFile(h, tempFile)

	err = h.Exec(fmt.Sprintf(`powershell -Command "$Input | Out-File -FilePath %s"`, ps.SingleQuote(tempFile)), exec.Stdin(data), exec.RedactString(data))
	if err != nil {
		return err
	}

	err = h.Exec(fmt.Sprintf(`powershell -Command "Move-Item -Force -Path %s -Destination %s"`, ps.SingleQuote(tempFile), ps.SingleQuote(path)))
	if err != nil {
		return err
	}

	return nil
}

// ReadFile reads a files contents from the host.
func (c Windows) ReadFile(h Host, path string) (string, error) {
	return h.ExecOutput(fmt.Sprintf(`type %s`, ps.DoubleQuote(path)), exec.HideOutput())
}

// DeleteFile deletes a file from the host.
func (c Windows) DeleteFile(h Host, path string) error {
	return h.Exec(fmt.Sprintf(`del /f %s`, ps.DoubleQuote(path)))
}

func (c Windows) deleteTempFile(h Host, path string) {
	if err := c.DeleteFile(h, path); err != nil {
		log.Debugf("failed to delete temporary file %s: %s", path, err.Error())
	}
}

// FileExist checks if a file exists on the host
func (c Windows) FileExist(h Host, path string) bool {
	return h.Exec(fmt.Sprintf(`powershell -Command "if (!(Test-Path -Path \"%s\")) { exit 1 }"`, path)) == nil
}

// UpdateEnvironment updates the hosts's environment variables
func (c Windows) UpdateEnvironment(h Host, env map[string]string) error {
	for k, v := range env {
		err := h.Exec(fmt.Sprintf(`setx %s %s`, ps.DoubleQuote(k), ps.DoubleQuote(v)))
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateServiceEnvironment does nothing on windows
func (c Windows) UpdateServiceEnvironment(_ Host, _ string, _ map[string]string) error {
	return nil
}

// CleanupEnvironment removes environment variable configuration
func (c Windows) CleanupEnvironment(h Host, env map[string]string) error {
	for k := range env {
		err := h.Exec(fmt.Sprintf(`powershell "[Environment]::SetEnvironmentVariable(%s, $null, 'User')"`, ps.SingleQuote(k)))
		if err != nil {
			return err
		}
		err = h.Exec(fmt.Sprintf(`powershell "[Environment]::SetEnvironmentVariable(%s, $null, 'Machine')"`, ps.SingleQuote(k)))
		if err != nil {
			return err
		}
	}
	return nil
}

// CleanupServiceEnvironment does nothing on windows
func (c Windows) CleanupServiceEnvironment(_ Host, _ string) error {
	return nil
}

// CommandExist returns true if the provided command exists
func (c Windows) CommandExist(h Host, cmd string) bool {
	return h.Execf("where /q %s", cmd) == nil
}

// Reboot executes the reboot command
func (c Windows) Reboot(h Host) error {
	return h.Exec("shutdown /r /t 5")
}

// StartService starts a a service
func (c Windows) StartService(h Host, s string) error {
	return h.Execf(`sc start "%s"`, s)
}

// StopService stops a a service
func (c Windows) StopService(h Host, s string) error {
	return h.Execf(`sc stop "%s"`, s)
}

// ServiceScriptPath returns the path to a service configuration file
func (c Windows) ServiceScriptPath(h Host, s string) (string, error) {
	return "", fmt.Errorf("not available on windows")
}

// RestartService restarts a a service
func (c Windows) RestartService(h Host, s string) error {
	return h.Execf(ps.Cmd(fmt.Sprintf(`Restart-Service "%s"`, s)))
}

// DaemonReload reloads init system configuration
func (c Windows) DaemonReload(_ Host) error {
	return nil
}

// EnableService enables a a service
func (c Windows) EnableService(h Host, s string) error {
	return h.Execf(`sc.exe config "%s" start=disabled`, s)
}

// DisableService disables a a service
func (c Windows) DisableService(h Host, s string) error {
	return h.Execf(`sc.exe config "%s" start=enabled`, s)
}

// ServiceIsRunning returns true if a service is running
func (c Windows) ServiceIsRunning(h Host, s string) bool {
	return h.Execf(`sc.exe query "%s" | findstr "RUNNING"`, s) == nil
}

// MkDir creates a directory (including intermediate directories)
func (c Windows) MkDir(h Host, s string, opts ...exec.Option) error {
	// windows mkdir is "-p" by default
	return h.Exec(fmt.Sprintf(`mkdir %s`, ps.DoubleQuote(s)), opts...)
}

// Chmod on windows does nothing
func (c Windows) Chmod(h Host, s, perm string, opts ...exec.Option) error {
	return nil
}

// Stat gets file / directory information
func (c Windows) Stat(h Host, path string, opts ...exec.Option) (*FileInfo, error) {
	f := &FileInfo{FName: path, FMode: fs.FileMode(0)}

	out, err := h.ExecOutput(fmt.Sprintf("[System.Math]::Truncate((Get-Date -Date ((Get-Item %s).LastWriteTime.ToUniversalTime()) -UFormat %%s))", ps.DoubleQuote(path)), opts...)
	if err != nil {
		return nil, err
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return nil, err
	}
	f.FModTime = time.Unix(ts, 0)

	out, err = h.ExecOutput(fmt.Sprintf("(Get-Item %s).Length", ps.DoubleQuote(path)), opts...)
	if err != nil {
		return nil, err
	}
	size, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return nil, err
	}
	f.FSize = size

	out, err = h.ExecOutput(fmt.Sprintf("(Get-Item %s).GetType().Name", ps.DoubleQuote(path)), opts...)
	if err != nil {
		return nil, err
	}
	f.FIsDir = strings.Contains(out, "DirectoryInfo")

	return f, nil
}

// Touch updates a file's last modified time or creates a new empty file
func (c Windows) Touch(h Host, path string, ts time.Time, opts ...exec.Option) error {
	if !c.FileExist(h, path) {
		if err := h.Exec(fmt.Sprintf("Set-Content -Path %s -value $null", ps.DoubleQuote(path)), opts...); err != nil {
			return err
		}
	}

	return h.Exec(fmt.Sprintf("(Get-Item %s).LastWriteTime = (Get-Date %s)", ps.DoubleQuote(path), ps.DoubleQuote(ts.Format(time.RFC3339))), opts...)
}

// LineIntoFile tries to find a line starting with the matcher and replace it with a new entry. If match isn't found, the string is appended to the file.
// TODO this is a straight copypaste from linux, figure out a way to share these
func (c Windows) LineIntoFile(h Host, path, matcher, newLine string) error {
	newLine = strings.TrimSuffix(newLine, "\n")
	content, err := c.ReadFile(h, path)
	if err != nil {
		content = ""
	}

	var found bool
	writer := new(strings.Builder)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		row := scanner.Text()

		if strings.HasPrefix(row, matcher) && !found {
			row = newLine
			found = true
		}

		fmt.Fprintln(writer, row)
	}

	if !found {
		fmt.Fprintln(writer, newLine)
	}

	return c.WriteFile(h, path, writer.String(), "0644")
}
