package initsystem

import (
	"fmt"
	"path"
	"strings"

	"github.com/k0sproject/rig/exec"
)

// Systemd is found by default on most linux distributions today
type Systemd struct{}

// StartService starts a a service
func (i Systemd) StartService(h Host, s string) error {
	return h.Execf("systemctl start %s 2> /dev/null", s, exec.Sudo(h))
}

// EnableService enables a a service
func (i Systemd) EnableService(h Host, s string) error {
	return h.Execf("systemctl enable %s 2> /dev/null", s, exec.Sudo(h))
}

// DisableService disables a a service
func (i Systemd) DisableService(h Host, s string) error {
	return h.Execf("systemctl disable %s 2> /dev/null", s, exec.Sudo(h))
}

// StopService stops a a service
func (i Systemd) StopService(h Host, s string) error {
	return h.Execf("systemctl stop %s 2> /dev/null", s, exec.Sudo(h))
}

// RestartService restarts a a service
func (i Systemd) RestartService(h Host, s string) error {
	return h.Execf("systemctl restart %s 2> /dev/null", s, exec.Sudo(h))
}

// DaemonReload reloads init system configuration
func (i Systemd) DaemonReload(h Host) error {
	return h.Execf("systemctl daemon-reload 2> /dev/null", exec.Sudo(h))
}

// ServiceIsRunning returns true if a service is running
func (i Systemd) ServiceIsRunning(h Host, s string) bool {
	return h.Execf(`systemctl status %s 2> /dev/null | grep -q "(running)"`, s, exec.Sudo(h)) == nil
}

// ServiceScriptPath returns the path to a service configuration file
func (i Systemd) ServiceScriptPath(h Host, s string) (string, error) {
	return h.ExecOutputf(`systemctl show -p FragmentPath %s.service 2> /dev/null | cut -d"=" -f2`, s, exec.Sudo(h))
}

// ServiceEnvironmentPath returns a path to an environment override file path
func (i Systemd) ServiceEnvironmentPath(h Host, s string) (string, error) {
	sp, err := i.ServiceScriptPath(h, s)
	if err != nil {
		return "", err
	}
	dn := path.Dir(sp)
	return path.Join(dn, fmt.Sprintf("%s.service.d", s), "env.conf"), nil
}

// ServiceEnvironmentContent returns a formatted string for a service environment override file
func (i Systemd) ServiceEnvironmentContent(env map[string]string) string {
	var b strings.Builder
	fmt.Fprintln(&b, "[Service]")
	for k, v := range env {
		_, _ = fmt.Fprintf(&b, "Environment=%s=%s\n", k, v)
	}

	return b.String()
}
