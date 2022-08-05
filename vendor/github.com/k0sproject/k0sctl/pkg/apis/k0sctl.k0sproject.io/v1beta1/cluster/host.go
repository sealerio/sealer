package cluster

import (
	"encoding/json"
	"fmt"
	gos "os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/creasty/defaults"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/go-playground/validator/v10"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/exec"
	"github.com/k0sproject/rig/os"
	"github.com/k0sproject/rig/os/registry"
	"github.com/k0sproject/version"
	log "github.com/sirupsen/logrus"
)

// Host contains all the needed details to work with hosts
type Host struct {
	rig.Connection `yaml:",inline"`

	Role             string            `yaml:"role"`
	PrivateInterface string            `yaml:"privateInterface,omitempty"`
	PrivateAddress   string            `yaml:"privateAddress,omitempty"`
	Environment      map[string]string `yaml:"environment,flow,omitempty"`
	UploadBinary     bool              `yaml:"uploadBinary,omitempty"`
	K0sBinaryPath    string            `yaml:"k0sBinaryPath,omitempty"`
	InstallFlags     Flags             `yaml:"installFlags,omitempty"`
	Files            []*UploadFile     `yaml:"files,omitempty"`
	OSIDOverride     string            `yaml:"os,omitempty"`
	HostnameOverride string            `yaml:"hostname,omitempty"`
	NoTaints         bool              `yaml:"noTaints,omitempty"`
	Hooks            Hooks             `yaml:"hooks,omitempty"`

	UploadBinaryPath string       `yaml:"-"`
	Metadata         HostMetadata `yaml:"-"`
	Configurer       configurer   `yaml:"-"`
}

func (h *Host) SetDefaults() {
	if h.OSIDOverride != "" {
		h.OSVersion = &rig.OSVersion{ID: h.OSIDOverride}
	}

	_ = defaults.Set(h.Connection)

	if h.InstallFlags.Get("--single") != "" && h.InstallFlags.GetValue("--single") != "false" && h.Role != "single" {
		log.Debugf("%s: changed role from '%s' to 'single' because of --single installFlag", h, h.Role)
		h.Role = "single"
	}
	if h.InstallFlags.Get("--enable-worker") != "" && h.InstallFlags.GetValue("--enable-worker") != "false" && h.Role != "controller+worker" {
		log.Debugf("%s: changed role from '%s' to 'controller+worker' because of --enable-worker installFlag", h, h.Role)
		h.Role = "controller+worker"
	}

	if h.InstallFlags.Get("--no-taints") != "" && h.InstallFlags.GetValue("--no-taints") != "false" {
		h.NoTaints = true
	}
}

func (h *Host) Validate() error {
	// For rig validation
	v := validator.New()
	if err := v.Struct(h); err != nil {
		return err
	}

	return validation.ValidateStruct(h,
		validation.Field(&h.Role, validation.In("controller", "worker", "controller+worker", "single").Error("unknown role "+h.Role)),
		validation.Field(&h.PrivateAddress, is.IP),
		validation.Field(&h.Files),
		validation.Field(&h.NoTaints, validation.When(h.Role != "controller+worker", validation.NotIn(true).Error("noTaints can only be true for controller+worker role"))),
	)
}

type configurer interface {
	Kind() string
	CheckPrivilege(os.Host) error
	StartService(os.Host, string) error
	StopService(os.Host, string) error
	RestartService(os.Host, string) error
	ServiceIsRunning(os.Host, string) bool
	Arch(os.Host) (string, error)
	K0sCmdf(string, ...interface{}) string
	K0sBinaryPath() string
	K0sBinaryVersion(os.Host) (*version.Version, error)
	K0sConfigPath() string
	K0sJoinTokenPath() string
	WriteFile(os.Host, string, string, string) error
	UpdateEnvironment(os.Host, map[string]string) error
	DaemonReload(os.Host) error
	ReplaceK0sTokenPath(os.Host, string) error
	ServiceScriptPath(os.Host, string) (string, error)
	ReadFile(os.Host, string) (string, error)
	FileExist(os.Host, string) bool
	Chmod(os.Host, string, string, ...exec.Option) error
	DownloadK0s(os.Host, *version.Version, string) error
	DownloadURL(os.Host, string, string, ...exec.Option) error
	InstallPackage(os.Host, ...string) error
	FileContains(os.Host, string, string) bool
	MoveFile(os.Host, string, string) error
	MkDir(os.Host, string, ...exec.Option) error
	DeleteFile(os.Host, string) error
	CommandExist(os.Host, string) bool
	Hostname(os.Host) string
	KubectlCmdf(string, ...interface{}) string
	KubeconfigPath() string
	IsContainer(os.Host) bool
	FixContainer(os.Host) error
	HTTPStatus(os.Host, string) (int, error)
	PrivateInterface(os.Host) (string, error)
	PrivateAddress(os.Host, string, string) (string, error)
	TempDir(os.Host) (string, error)
	TempFile(os.Host) (string, error)
	UpdateServiceEnvironment(os.Host, string, map[string]string) error
	CleanupServiceEnvironment(os.Host, string) error
	Stat(os.Host, string, ...exec.Option) (*os.FileInfo, error)
	Touch(os.Host, string, time.Time, ...exec.Option) error
	DeleteDir(os.Host, string, ...exec.Option) error
	K0sctlLockFilePath(os.Host) string
	UpsertFile(os.Host, string, string) error
}

// HostMetadata resolved metadata for host
type HostMetadata struct {
	K0sBinaryVersion  string
	K0sRunningVersion string
	Arch              string
	IsK0sLeader       bool
	Hostname          string
	Ready             bool
	NeedsUpgrade      bool
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (h *Host) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type host Host
	yh := (*host)(h)

	yh.Environment = make(map[string]string)

	if err := unmarshal(yh); err != nil {
		return err
	}

	return defaults.Set(h)
}

// Address returns an address for the host
func (h *Host) Address() string {
	if h.SSH != nil {
		return h.SSH.Address
	}

	if h.WinRM != nil {
		return h.WinRM.Address
	}

	return "127.0.0.1"
}

// Protocol returns host communication protocol
func (h *Host) Protocol() string {
	if h.SSH != nil {
		return "ssh"
	}

	if h.WinRM != nil {
		return "winrm"
	}

	if h.Localhost != nil {
		return "local"
	}

	return "nil"
}

// ResolveConfigurer assigns a rig-style configurer to the Host (see configurer/)
func (h *Host) ResolveConfigurer() error {
	bf, err := registry.GetOSModuleBuilder(*h.OSVersion)
	if err != nil {
		return err
	}

	if c, ok := bf().(configurer); ok {
		h.Configurer = c

		return nil
	}

	return fmt.Errorf("unsupported OS")
}

// K0sJoinTokenPath returns the token file path from install flags or configurer
func (h *Host) K0sJoinTokenPath() string {
	if path := h.InstallFlags.GetValue("--token-file"); path != "" {
		return path
	}

	return h.Configurer.K0sJoinTokenPath()
}

// K0sConfigPath returns the config file path from install flags or configurer
func (h *Host) K0sConfigPath() string {
	if path := h.InstallFlags.GetValue("--config"); path != "" {
		return path
	}

	if path := h.InstallFlags.GetValue("-c"); path != "" {
		return path
	}

	return h.Configurer.K0sConfigPath()
}

// unquote + unescape a string
func unQE(s string) string {
	unq, err := strconv.Unquote(s)
	if err != nil {
		return s
	}

	c := string(s[0])                                           // string was quoted, c now has the quote char
	re := regexp.MustCompile(fmt.Sprintf(`(?:^|[^\\])\\%s`, c)) // replace \" with " (remove escaped quotes inside quoted string)
	return string(re.ReplaceAllString(unq, c))
}

// K0sInstallCommand returns a full command that will install k0s service with necessary flags
func (h *Host) K0sInstallCommand() string {
	role := h.Role
	flags := h.InstallFlags

	switch role {
	case "controller+worker":
		role = "controller"
		flags.AddUnlessExist("--enable-worker")
		if h.NoTaints {
			flags.AddUnlessExist("--no-taints")
		}
	case "single":
		role = "controller"
		flags.AddUnlessExist("--single")
	}

	if !h.Metadata.IsK0sLeader {
		flags.AddUnlessExist(fmt.Sprintf(`--token-file "%s"`, h.K0sJoinTokenPath()))
	}

	if h.IsController() {
		flags.AddUnlessExist(fmt.Sprintf(`--config "%s"`, h.K0sConfigPath()))
	}

	if strings.HasSuffix(h.Role, "worker") {
		var extra Flags
		if old := flags.GetValue("--kubelet-extra-args"); old != "" {
			extra = Flags{unQE(old)}
		}
		// set worker's private address to --node-ip in --extra-kubelet-args
		if h.PrivateAddress != "" {
			extra.AddUnlessExist(fmt.Sprintf("--node-ip=%s", h.PrivateAddress))
		}
		if h.HostnameOverride != "" {
			extra.AddOrReplace(fmt.Sprintf("--hostname-override=%s", h.HostnameOverride))
		}
		if extra != nil {
			flags.AddOrReplace(fmt.Sprintf("--kubelet-extra-args=%s", strconv.Quote(extra.Join())))
		}
	}

	cmd := h.Configurer.K0sCmdf("install %s %s", role, flags.Join())
	sudocmd, err := h.Sudo(cmd)
	if err != nil {
		log.Warnf("%s: %s", h, err.Error())
		return cmd
	}
	return sudocmd
}

// K0sBackupCommand returns a full command to be used as run k0s backup
func (h *Host) K0sBackupCommand(targetDir string) string {
	return h.Configurer.K0sCmdf("backup --save-path %s", targetDir)
}

// K0sRestoreCommand returns a full command to restore cluster state from a backup
func (h *Host) K0sRestoreCommand(backupfile string) string {
	return h.Configurer.K0sCmdf("restore %s", backupfile)
}

// IsController returns true for controller and controller+worker roles
func (h *Host) IsController() bool {
	return h.Role == "controller" || h.Role == "controller+worker" || h.Role == "single"
}

// K0sServiceName returns correct service name
func (h *Host) K0sServiceName() string {
	switch h.Role {
	case "controller", "controller+worker", "single":
		return "k0scontroller"
	default:
		return "k0sworker"
	}
}

// UpdateK0sBinary updates the binary on the host either by downloading or uploading, based on the config
func (h *Host) UpdateK0sBinary(version *version.Version) error {
	if h.UploadBinaryPath != "" {
		if err := h.Upload(h.UploadBinaryPath, h.Configurer.K0sBinaryPath(), exec.Sudo(h)); err != nil {
			return err
		}
		if err := h.Configurer.Chmod(h, h.Configurer.K0sBinaryPath(), "0700", exec.Sudo(h)); err != nil {
			return err
		}
	} else {
		if err := h.Configurer.DownloadK0s(h, version, h.Metadata.Arch); err != nil {
			return err
		}
	}

	updatedVersion, err := h.Configurer.K0sBinaryVersion(h)
	if err != nil {
		return fmt.Errorf("failed to get updated k0s binary version: %w", err)
	}
	if !version.Equal(updatedVersion) {
		return fmt.Errorf("updated k0s binary version is %s not %s", updatedVersion, version)
	}

	h.Metadata.K0sBinaryVersion = version.String()

	return nil
}

type kubeNodeStatus struct {
	Items []struct {
		Status struct {
			Conditions []struct {
				Status string `json:"status"`
				Type   string `json:"type"`
			} `json:"conditions"`
		} `json:"status"`
	} `json:"items"`
}

// KubeNodeReady runs kubectl on the host and returns true if the given node is marked as ready
func (h *Host) KubeNodeReady(node *Host) (bool, error) {
	output, err := h.ExecOutput(h.Configurer.KubectlCmdf("get node -l kubernetes.io/hostname=%s -o json", node.Metadata.Hostname), exec.HideOutput(), exec.Sudo(h))
	if err != nil {
		return false, err
	}
	log.Tracef("node status output:\n%s\n", output)
	status := kubeNodeStatus{}
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		return false, fmt.Errorf("failed to decode kubectl output: %s", err.Error())
	}
	for _, i := range status.Items {
		for _, c := range i.Status.Conditions {
			log.Debugf("%s: node status condition %s = %s", node, c.Type, c.Status)
			if c.Type == "Ready" {
				return c.Status == "True", nil
			}
		}
	}

	log.Debugf("%s: failed to find Ready=True state in kubectl output", node)
	return false, nil
}

// WaitKubeNodeReady blocks until node becomes ready. TODO should probably use Context
func (h *Host) WaitKubeNodeReady(node *Host) error {
	return retry.Do(
		func() error {
			status, err := h.KubeNodeReady(node)
			if err != nil {
				return err
			}
			if !status {
				return fmt.Errorf("%s: node %s status not reported as ready", h, node.Metadata.Hostname)
			}
			return nil
		},
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(120),
		retry.LastErrorOnly(true),
	)
}

// DrainNode drains the given node
func (h *Host) DrainNode(node *Host) error {
	return h.Exec(h.Configurer.KubectlCmdf("drain --grace-period=120 --force --timeout=5m --ignore-daemonsets --delete-local-data %s", node.Metadata.Hostname), exec.Sudo(h))
}

// UncordonNode marks the node schedulable again
func (h *Host) UncordonNode(node *Host) error {
	return h.Exec(h.Configurer.KubectlCmdf("uncordon %s", node.Metadata.Hostname), exec.Sudo(h))
}

// CheckHTTPStatus will perform a web request to the url and return an error if the http status is not the expected
func (h *Host) CheckHTTPStatus(url string, expected ...int) error {
	status, err := h.Configurer.HTTPStatus(h, url)
	if err != nil {
		return err
	}

	for _, e := range expected {
		if status == e {
			return nil
		}
	}

	return fmt.Errorf("expected response code %d but received %d", expected, status)

}

// WaitHTTPStatus waits until http status received for a GET from the URL is the expected one
func (h *Host) WaitHTTPStatus(url string, expected ...int) error {
	return retry.Do(
		func() error {
			return h.CheckHTTPStatus(url, expected...)
		},
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(60),
		retry.LastErrorOnly(true),
	)
}

// WaitK0sServiceRunning blocks until the k0s service is running on the host
func (h *Host) WaitK0sServiceRunning() error {
	return retry.Do(
		func() error {
			if !h.Configurer.ServiceIsRunning(h, h.K0sServiceName()) {
				return fmt.Errorf("not running")
			}
			return h.Exec(h.Configurer.K0sCmdf("status"), exec.Sudo(h))
		},
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(60),
		retry.LastErrorOnly(true),
	)
}

// WaitK0sServiceStopped blocks until the k0s service is no longer running on the host
func (h *Host) WaitK0sServiceStopped() error {
	return retry.Do(
		func() error {
			if h.Configurer.ServiceIsRunning(h, h.K0sServiceName()) {
				return fmt.Errorf("k0s still running")
			}
			if h.Exec(h.Configurer.K0sCmdf("status"), exec.Sudo(h)) == nil {
				return fmt.Errorf("k0s still running")
			}
			return nil
		},
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(60),
		retry.LastErrorOnly(true),
	)
}

// NeedCurl returns true when the curl package is needed on the host
func (h *Host) NeedCurl() bool {
	// Windows does not need any packages for web requests
	if h.Configurer.Kind() == "windows" {
		return false
	}

	return !h.Configurer.CommandExist(h, "curl")
}

// NeedIPTables returns true when the iptables package is needed on the host
func (h *Host) NeedIPTables() bool {
	// Windows does not need iptables
	if h.Configurer.Kind() == "windows" {
		return false
	}

	// Controllers do not need iptables
	if h.IsController() {
		return false
	}

	return !h.Configurer.CommandExist(h, "iptables")
}

// NeedInetUtils returns true when the inetutils package is needed on the host to run `hostname`.
func (h *Host) NeedInetUtils() bool {
	// Windows does not need inetutils
	if h.Configurer.Kind() == "windows" {
		return false
	}

	return !h.Configurer.CommandExist(h, "hostname")
}

// WaitKubeAPIReady blocks until the local kube api responds to /version
func (h *Host) WaitKubeAPIReady(port int) error {
	// If the anon-auth is disabled on kube api the version endpoint will give 401
	// thus we need to accept both 200 and 401 as valid statuses when checking kube api
	return h.WaitHTTPStatus(fmt.Sprintf("https://localhost:%d/version", port), 200, 401)
}

// FileChanged returns true when a remote file has different size or mtime compared to local
// or if an error occurs
func (h *Host) FileChanged(lpath, rpath string) bool {
	lstat, err := gos.Stat(lpath)
	if err != nil {
		log.Debugf("%s: local stat failed: %s", h, err)
		return true
	}
	rstat, err := h.Configurer.Stat(h, rpath, exec.Sudo(h))
	if err != nil {
		log.Debugf("%s: remote stat failed: %s", h, err)
		return true
	}

	if lstat.Size() != rstat.Size() {
		log.Debugf("%s: file sizes for %s differ (%d vs %d)", h, lpath, lstat.Size(), rstat.Size())
		return true
	}

	if !lstat.ModTime().Equal(rstat.ModTime()) {
		log.Debugf("%s: file modtimes for %s differ (%s vs %s)", h, lpath, lstat.ModTime(), rstat.ModTime())
		return true
	}

	return false
}
