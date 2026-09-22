//go:build windows
// +build windows

package platform

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// newPlatform creates a new platform instance for Windows
func newPlatform() Platform {
	return newWindowsPlatform()
}

// windowsPlatform implements Platform for Windows systems
type windowsPlatform struct {
	paths   *PathConfig
	service *windowsServiceManager
	command *windowsCommandExecutor
	fs      *windowsFileSystem
}

// newWindowsPlatform creates a new Windows platform instance
func newWindowsPlatform() Platform {
	p := &windowsPlatform{
		paths: &PathConfig{
			// Container runtime paths
			ContainerdBinDir:    `C:\Program Files\containerd\bin`,
			ContainerdConfigDir: `C:\Program Files\containerd`,
			ContainerdDataDir:   `C:\ProgramData\containerd`,
			ContainerdSocketDir: `\\.\pipe`,

			// Kubernetes paths
			KubeletBinDir:     `C:\k`,
			KubeletConfigDir:  `C:\etc\kubernetes`,
			KubeletDataDir:    `C:\var\lib\kubelet`,
			KubeletManifests:  `C:\etc\kubernetes\manifests`,
			KubeletVolumeDir:  `C:\etc\kubernetes\volumeplugins`,
			KubeletServiceDir: `C:\etc\kubernetes\kubelet.conf.d`,

			// CNI paths
			CNIBinDir:  `C:\opt\cni\bin`,
			CNIConfDir: `C:\etc\cni\net.d`,

			// System paths
			SystemBinDir:    `C:\Windows\System32`,
			SystemConfigDir: `C:\ProgramData`,
			SystemDataDir:   `C:\ProgramData`,
			SystemLogDir:    `C:\var\log`,
			TempDir:         os.TempDir(),

			// Service paths (Windows uses SCM, no service files)
			ServiceDir:     "",
			ServiceConfDir: `C:\ProgramData\aks-flex-node`,

			// Azure Arc paths
			ArcAgentBinDir:  `C:\Program Files\AzureConnectedMachineAgent`,
			ArcAgentDataDir: `C:\ProgramData\AzureConnectedMachineAgent`,

			// File extensions
			ExecutableExt: ".exe",
			ArchiveExt:    ".zip",
			ServiceExt:    "",
		},
	}
	p.service = &windowsServiceManager{}
	p.command = &windowsCommandExecutor{}
	p.fs = &windowsFileSystem{}
	return p
}

func (p *windowsPlatform) OS() OS {
	return Windows
}

func (p *windowsPlatform) Paths() *PathConfig {
	return p.paths
}

func (p *windowsPlatform) Service() ServiceManager {
	return p.service
}

func (p *windowsPlatform) Command() CommandExecutor {
	return p.command
}

func (p *windowsPlatform) FileSystem() FileSystem {
	return p.fs
}

// windowsCommandExecutor implements CommandExecutor for Windows
type windowsCommandExecutor struct{}

func (e *windowsCommandExecutor) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (e *windowsCommandExecutor) RunWithOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// On Windows, we typically run as Administrator, so privileged == regular
func (e *windowsCommandExecutor) RunPrivileged(ctx context.Context, name string, args ...string) error {
	return e.Run(ctx, name, args...)
}

func (e *windowsCommandExecutor) RunPrivilegedWithOutput(ctx context.Context, name string, args ...string) (string, error) {
	return e.RunWithOutput(ctx, name, args...)
}

// windowsFileSystem implements FileSystem for Windows
type windowsFileSystem struct{}

func (fs *windowsFileSystem) FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (fs *windowsFileSystem) DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func (fs *windowsFileSystem) CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

func (fs *windowsFileSystem) WriteFile(path string, content []byte, perm uint32) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, content, os.FileMode(perm))
}

func (fs *windowsFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (fs *windowsFileSystem) RemoveFile(path string) error {
	return os.Remove(path)
}

func (fs *windowsFileSystem) RemoveDirectory(path string) error {
	return os.RemoveAll(path)
}

func (fs *windowsFileSystem) DownloadFile(url, destination string) error {
	client := &http.Client{Timeout: 10 * time.Minute}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d for %s", resp.StatusCode, url)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(destination)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
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

func (fs *windowsFileSystem) ExtractTarGz(archive, destination string) error {
	// Windows has tar built-in since Windows 10 1803
	return exec.Command("tar", "-C", destination, "-xzf", archive).Run()
}

func (fs *windowsFileSystem) GetArchitecture() (string, error) {
	// On Windows, check PROCESSOR_ARCHITECTURE environment variable
	arch := os.Getenv("PROCESSOR_ARCHITECTURE")
	switch strings.ToLower(arch) {
	case "amd64", "x64":
		return "amd64", nil
	case "arm64":
		return "arm64", nil
	case "x86":
		return "386", nil
	default:
		return arch, nil
	}
}

// windowsServiceManager implements ServiceManager for Windows using SCM
type windowsServiceManager struct{}

func (s *windowsServiceManager) Install(config *ServiceConfig) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Build the binary path with properly escaped arguments
	binaryPath := config.BinaryPath
	if len(config.Args) > 0 {
		// Quote arguments that contain spaces or special characters
		var quotedArgs []string
		for _, arg := range config.Args {
			// If arg contains spaces, quotes, or special chars, quote it
			if strings.ContainsAny(arg, " \t\"") {
				// Escape any existing quotes and wrap in quotes
				escaped := strings.ReplaceAll(arg, `"`, `\"`)
				quotedArgs = append(quotedArgs, `"`+escaped+`"`)
			} else {
				quotedArgs = append(quotedArgs, arg)
			}
		}
		binaryPath = binaryPath + " " + strings.Join(quotedArgs, " ")
	}

	// Determine start type
	startType := uint32(mgr.StartAutomatic)

	// Create service configuration
	svcConfig := mgr.Config{
		DisplayName:  config.DisplayName,
		Description:  config.Description,
		StartType:    startType,
		Dependencies: config.Dependencies,
	}

	// Create the service
	svc, err := m.CreateService(config.Name, binaryPath, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service %s: %w", config.Name, err)
	}
	defer svc.Close()

	// Configure recovery options (restart on failure)
	if config.RestartPolicy == RestartAlways || config.RestartPolicy == RestartOnFailure {
		recoveryActions := []mgr.RecoveryAction{
			{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
			{Type: mgr.ServiceRestart, Delay: 15 * time.Second},
			{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
		}
		if err := svc.SetRecoveryActions(recoveryActions, 30); err != nil {
			// Log but don't fail - recovery actions are optional
			fmt.Printf("Warning: failed to set recovery actions: %v\n", err)
		}
	}

	return nil
}

func (s *windowsServiceManager) Uninstall(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	svc, err := m.OpenService(name)
	if err != nil {
		// Service doesn't exist, that's fine for uninstall
		return nil
	}
	defer svc.Close()

	// Stop the service first
	_ = s.Stop(name)

	// Delete the service
	if err := svc.Delete(); err != nil {
		return fmt.Errorf("failed to delete service %s: %w", name, err)
	}

	return nil
}

func (s *windowsServiceManager) Start(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	svc, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("failed to open service %s: %w", name, err)
	}
	defer svc.Close()

	return svc.Start()
}

func (s *windowsServiceManager) Stop(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	service, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("failed to open service %s: %w", name, err)
	}
	defer service.Close()

	_, err = service.Control(svc.Stop)
	return err
}

func (s *windowsServiceManager) Restart(name string) error {
	if err := s.Stop(name); err != nil {
		// Continue even if stop fails (service might not be running)
	}

	// Wait a moment for the service to fully stop
	time.Sleep(2 * time.Second)

	return s.Start(name)
}

func (s *windowsServiceManager) Enable(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	svc, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("failed to open service %s: %w", name, err)
	}
	defer svc.Close()

	config, err := svc.Config()
	if err != nil {
		return fmt.Errorf("failed to get service config: %w", err)
	}

	config.StartType = mgr.StartAutomatic
	return svc.UpdateConfig(config)
}

func (s *windowsServiceManager) Disable(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	svc, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("failed to open service %s: %w", name, err)
	}
	defer svc.Close()

	config, err := svc.Config()
	if err != nil {
		return fmt.Errorf("failed to get service config: %w", err)
	}

	config.StartType = mgr.StartDisabled
	return svc.UpdateConfig(config)
}

func (s *windowsServiceManager) IsActive(name string) bool {
	m, err := mgr.Connect()
	if err != nil {
		return false
	}
	defer m.Disconnect()

	service, err := m.OpenService(name)
	if err != nil {
		return false
	}
	defer service.Close()

	status, err := service.Query()
	if err != nil {
		return false
	}

	return status.State == svc.Running
}

func (s *windowsServiceManager) IsEnabled(name string) bool {
	m, err := mgr.Connect()
	if err != nil {
		return false
	}
	defer m.Disconnect()

	svc, err := m.OpenService(name)
	if err != nil {
		return false
	}
	defer svc.Close()

	config, err := svc.Config()
	if err != nil {
		return false
	}

	return config.StartType == mgr.StartAutomatic
}

func (s *windowsServiceManager) Exists(name string) bool {
	m, err := mgr.Connect()
	if err != nil {
		return false
	}
	defer m.Disconnect()

	svc, err := m.OpenService(name)
	if err != nil {
		return false
	}
	svc.Close()
	return true
}

func (s *windowsServiceManager) WaitForService(name string, timeoutSeconds int) error {
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

func (s *windowsServiceManager) ReloadDaemon() error {
	// Windows SCM doesn't need explicit reload like systemd
	return nil
}
