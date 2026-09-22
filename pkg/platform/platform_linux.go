//go:build linux
// +build linux

package platform

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// newPlatform creates a new platform instance for Linux
func newPlatform() Platform {
	return newLinuxPlatform()
}

// linuxPlatform implements Platform for Linux systems
type linuxPlatform struct {
	paths   *PathConfig
	service *linuxServiceManager
	command *linuxCommandExecutor
	fs      *linuxFileSystem
}

// newLinuxPlatform creates a new Linux platform instance
func newLinuxPlatform() Platform {
	p := &linuxPlatform{
		paths: &PathConfig{
			// Container runtime paths
			ContainerdBinDir:    "/usr/bin",
			ContainerdConfigDir: "/etc/containerd",
			ContainerdDataDir:   "/var/lib/containerd",
			ContainerdSocketDir: "/run/containerd",

			// Kubernetes paths
			KubeletBinDir:     "/usr/local/bin",
			KubeletConfigDir:  "/etc/kubernetes",
			KubeletDataDir:    "/var/lib/kubelet",
			KubeletManifests:  "/etc/kubernetes/manifests",
			KubeletVolumeDir:  "/etc/kubernetes/volumeplugins",
			KubeletServiceDir: "/etc/systemd/system/kubelet.service.d",

			// CNI paths
			CNIBinDir:  "/opt/cni/bin",
			CNIConfDir: "/etc/cni/net.d",

			// System paths
			SystemBinDir:    "/usr/bin",
			SystemConfigDir: "/etc",
			SystemDataDir:   "/var/lib",
			SystemLogDir:    "/var/log",
			TempDir:         "/tmp",

			// Service paths
			ServiceDir:     "/etc/systemd/system",
			ServiceConfDir: "/etc/default",

			// Azure Arc paths
			ArcAgentBinDir:  "/usr/bin",
			ArcAgentDataDir: "/var/lib/waagent",

			// File extensions
			ExecutableExt: "",
			ArchiveExt:    ".tar.gz",
			ServiceExt:    ".service",
		},
	}
	p.service = &linuxServiceManager{}
	p.command = &linuxCommandExecutor{}
	p.fs = &linuxFileSystem{}
	return p
}

func (p *linuxPlatform) OS() OS {
	return Linux
}

func (p *linuxPlatform) Paths() *PathConfig {
	return p.paths
}

func (p *linuxPlatform) Service() ServiceManager {
	return p.service
}

func (p *linuxPlatform) Command() CommandExecutor {
	return p.command
}

func (p *linuxPlatform) FileSystem() FileSystem {
	return p.fs
}

// linuxCommandExecutor implements CommandExecutor for Linux
type linuxCommandExecutor struct{}

// Commands that always need sudo
var alwaysNeedsSudo = []string{"apt", "apt-get", "dpkg", "systemctl", "mount", "umount", "modprobe", "sysctl", "azcmagent", "usermod", "kubectl"}

// Commands that need sudo based on path
var conditionalSudo = []string{"mkdir", "cp", "chmod", "chown", "mv", "tar", "rm", "bash", "install", "ln", "cat"}

// System paths that require elevated privileges
var systemPaths = []string{"/etc/", "/usr/", "/var/", "/opt/", "/boot/", "/sys/"}

func (e *linuxCommandExecutor) requiresSudo(name string, args []string) bool {
	// Check if already running as root
	if os.Geteuid() == 0 {
		return false
	}

	// Check if this command always needs sudo
	for _, sudoCmd := range alwaysNeedsSudo {
		if name == sudoCmd {
			return true
		}
	}

	// Check if this command needs sudo based on the paths involved
	for _, sudoCmd := range conditionalSudo {
		if name == sudoCmd {
			for _, arg := range args {
				for _, sysPath := range systemPaths {
					if strings.HasPrefix(arg, sysPath) {
						return true
					}
				}
			}
			break
		}
	}

	return false
}

func (e *linuxCommandExecutor) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (e *linuxCommandExecutor) RunWithOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (e *linuxCommandExecutor) RunPrivileged(ctx context.Context, name string, args ...string) error {
	if e.requiresSudo(name, args) {
		allArgs := append([]string{"-E", name}, args...)
		cmd := exec.CommandContext(ctx, "sudo", allArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return e.Run(ctx, name, args...)
}

func (e *linuxCommandExecutor) RunPrivilegedWithOutput(ctx context.Context, name string, args ...string) (string, error) {
	if e.requiresSudo(name, args) {
		allArgs := append([]string{"-E", name}, args...)
		cmd := exec.CommandContext(ctx, "sudo", allArgs...)
		output, err := cmd.CombinedOutput()
		return string(output), err
	}
	return e.RunWithOutput(ctx, name, args...)
}

// linuxFileSystem implements FileSystem for Linux
type linuxFileSystem struct{}

func (fs *linuxFileSystem) FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (fs *linuxFileSystem) DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func (fs *linuxFileSystem) CreateDirectory(path string) error {
	cmd := exec.Command("mkdir", "-p", path)
	// Check if it needs sudo
	for _, sysPath := range systemPaths {
		if strings.HasPrefix(path, sysPath) && os.Geteuid() != 0 {
			cmd = exec.Command("sudo", "mkdir", "-p", path)
			break
		}
	}
	return cmd.Run()
}

func (fs *linuxFileSystem) WriteFile(path string, content []byte, perm uint32) error {
	// For system paths, use temp file + sudo mv approach
	needsSudo := false
	for _, sysPath := range systemPaths {
		if strings.HasPrefix(path, sysPath) && os.Geteuid() != 0 {
			needsSudo = true
			break
		}
	}

	if needsSudo {
		// Create temp file
		tmpFile, err := os.CreateTemp("", "aks-flex-node-*")
		if err != nil {
			return err
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		if _, err := tmpFile.Write(content); err != nil {
			tmpFile.Close()
			return err
		}
		tmpFile.Close()

		// Copy to destination using sudo
		if err := exec.Command("sudo", "cp", tmpPath, path).Run(); err != nil {
			return err
		}

		// Set permissions
		return exec.Command("sudo", "chmod", fmt.Sprintf("%o", perm), path).Run()
	}

	return os.WriteFile(path, content, os.FileMode(perm))
}

func (fs *linuxFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (fs *linuxFileSystem) RemoveFile(path string) error {
	// Check if sudo needed
	for _, sysPath := range systemPaths {
		if strings.HasPrefix(path, sysPath) && os.Geteuid() != 0 {
			return exec.Command("sudo", "rm", "-f", path).Run()
		}
	}
	return os.Remove(path)
}

func (fs *linuxFileSystem) RemoveDirectory(path string) error {
	// Check if sudo needed
	for _, sysPath := range systemPaths {
		if strings.HasPrefix(path, sysPath) && os.Geteuid() != 0 {
			return exec.Command("sudo", "rm", "-rf", path).Run()
		}
	}
	return os.RemoveAll(path)
}

func (fs *linuxFileSystem) DownloadFile(url, destination string) error {
	client := &http.Client{Timeout: 10 * time.Minute}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d for %s", resp.StatusCode, url)
	}

	out, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destination, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", destination, err)
	}

	return nil
}

func (fs *linuxFileSystem) ExtractTarGz(archive, destination string) error {
	return exec.Command("tar", "-C", destination, "-xzf", archive).Run()
}

func (fs *linuxFileSystem) GetArchitecture() (string, error) {
	output, err := exec.Command("uname", "-m").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get architecture: %w", err)
	}

	arch := strings.TrimSpace(string(output))
	switch arch {
	case "x86_64":
		return "amd64", nil
	case "aarch64":
		return "arm64", nil
	case "armv7l", "armv7":
		return "arm", nil
	default:
		return arch, nil
	}
}

// linuxServiceManager implements ServiceManager for Linux using systemd
type linuxServiceManager struct{}

func (s *linuxServiceManager) Install(config *ServiceConfig) error {
	// Generate systemd unit file content
	unitContent := s.generateUnitFile(config)

	// Write unit file
	unitPath := fmt.Sprintf("/etc/systemd/system/%s.service", config.Name)

	// Create temp file and copy with sudo
	tmpFile, err := os.CreateTemp("", "systemd-unit-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(unitContent); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Copy to systemd directory
	if err := exec.Command("sudo", "cp", tmpPath, unitPath).Run(); err != nil {
		return fmt.Errorf("failed to install service file: %w", err)
	}

	// Set permissions
	if err := exec.Command("sudo", "chmod", "644", unitPath).Run(); err != nil {
		return fmt.Errorf("failed to set service file permissions: %w", err)
	}

	// Reload systemd
	return s.ReloadDaemon()
}

func (s *linuxServiceManager) generateUnitFile(config *ServiceConfig) string {
	var sb strings.Builder

	// [Unit] section
	sb.WriteString("[Unit]\n")
	sb.WriteString(fmt.Sprintf("Description=%s\n", config.Description))
	if len(config.Dependencies) > 0 {
		sb.WriteString(fmt.Sprintf("After=%s\n", strings.Join(config.Dependencies, " ")))
		sb.WriteString(fmt.Sprintf("Requires=%s\n", strings.Join(config.Dependencies, " ")))
	}

	// [Service] section
	sb.WriteString("\n[Service]\n")

	// Build ExecStart
	execStart := config.BinaryPath
	if len(config.Args) > 0 {
		execStart += " " + strings.Join(config.Args, " ")
	}
	sb.WriteString(fmt.Sprintf("ExecStart=%s\n", execStart))

	if config.WorkingDir != "" {
		sb.WriteString(fmt.Sprintf("WorkingDirectory=%s\n", config.WorkingDir))
	}

	if config.User != "" {
		sb.WriteString(fmt.Sprintf("User=%s\n", config.User))
	}

	// Restart policy
	switch config.RestartPolicy {
	case RestartAlways:
		sb.WriteString("Restart=always\n")
	case RestartOnFailure:
		sb.WriteString("Restart=on-failure\n")
	case RestartNever:
		sb.WriteString("Restart=no\n")
	default:
		sb.WriteString("Restart=always\n")
	}

	if config.RestartDelayMs > 0 {
		sb.WriteString(fmt.Sprintf("RestartSec=%d\n", config.RestartDelayMs/1000))
	}

	// Environment variables
	for k, v := range config.Environment {
		sb.WriteString(fmt.Sprintf("Environment=%s=%s\n", k, v))
	}

	// [Install] section
	sb.WriteString("\n[Install]\n")
	sb.WriteString("WantedBy=multi-user.target\n")

	return sb.String()
}

func (s *linuxServiceManager) Uninstall(name string) error {
	// Stop and disable first
	_ = s.Stop(name)
	_ = s.Disable(name)

	// Remove unit file
	unitPath := fmt.Sprintf("/etc/systemd/system/%s.service", name)
	if err := exec.Command("sudo", "rm", "-f", unitPath).Run(); err != nil {
		return err
	}

	// Remove drop-in directory if exists
	dropInDir := fmt.Sprintf("/etc/systemd/system/%s.service.d", name)
	_ = exec.Command("sudo", "rm", "-rf", dropInDir).Run()

	return s.ReloadDaemon()
}

func (s *linuxServiceManager) Start(name string) error {
	return exec.Command("sudo", "systemctl", "start", name).Run()
}

func (s *linuxServiceManager) Stop(name string) error {
	return exec.Command("sudo", "systemctl", "stop", name).Run()
}

func (s *linuxServiceManager) Restart(name string) error {
	return exec.Command("sudo", "systemctl", "restart", name).Run()
}

func (s *linuxServiceManager) Enable(name string) error {
	return exec.Command("sudo", "systemctl", "enable", name).Run()
}

func (s *linuxServiceManager) Disable(name string) error {
	return exec.Command("sudo", "systemctl", "disable", name).Run()
}

func (s *linuxServiceManager) IsActive(name string) bool {
	output, err := exec.Command("systemctl", "is-active", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "active"
}

func (s *linuxServiceManager) IsEnabled(name string) bool {
	output, err := exec.Command("systemctl", "is-enabled", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "enabled"
}

func (s *linuxServiceManager) Exists(name string) bool {
	err := exec.Command("systemctl", "list-unit-files", name+".service").Run()
	return err == nil
}

func (s *linuxServiceManager) WaitForService(name string, timeoutSeconds int) error {
	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		if s.IsActive(name) {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for service %s to start", name)
		}

		<-ticker.C
	}
}

func (s *linuxServiceManager) ReloadDaemon() error {
	return exec.Command("sudo", "systemctl", "daemon-reload").Run()
}
