package containerd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
	"go.goms.io/aks/AKSFlexNode/pkg/platform"
	"go.goms.io/aks/AKSFlexNode/pkg/utils"
)

// Installer handles containerd installation operations
type Installer struct {
	config   *config.Config
	logger   *logrus.Logger
	platform platform.Platform
}

// NewInstaller creates a new containerd Installer
func NewInstaller(logger *logrus.Logger) *Installer {
	return &Installer{
		config:   config.GetConfig(),
		logger:   logger,
		platform: platform.Current(),
	}
}

// Execute downloads and installs the containerd container runtime with required plugins
func (i *Installer) Execute(ctx context.Context) error {
	i.logger.Info("Step 1: Preparing containerd directories")
	if err := i.prepareContainerdDirectories(); err != nil {
		return fmt.Errorf("failed to prepare containerd directories: %w", err)
	}
	i.logger.Info("Prepared containerd directories successfully")

	i.logger.Infof("Step 2: Downloading and installing containerd version %s", i.getContainerdVersion())
	if err := i.installContainerd(); err != nil {
		return fmt.Errorf("failed to install containerd: %w", err)
	}
	i.logger.Info("containerd binaries installed successfully")

	// Configure containerd service and configuration files
	i.logger.Info("Step 3: Configuring containerd")
	if err := i.configure(); err != nil {
		return fmt.Errorf("containerd configuration failed: %w", err)
	}
	i.logger.Info("containerd configured successfully")

	i.logger.Info("Installer: containerd installed and configured successfully")
	return nil
}

func (i *Installer) prepareContainerdDirectories() error {
	fs := i.platform.FileSystem()

	for _, dir := range containerdDirs {
		// Create directory if it doesn't exist
		if !fs.DirectoryExists(dir) {
			if err := fs.CreateDirectory(dir); err != nil {
				return fmt.Errorf("failed to create containerd directory %s: %w", dir, err)
			}
		}

		// Clean up any existing configurations to start fresh (config dir only)
		if dir == defaultContainerdConfigDir {
			i.logger.Debugf("Cleaning existing containerd configurations in: %s", dir)
			if platform.IsLinux() {
				if err := utils.RunSystemCommand("rm", "-rf", dir+"/*"); err != nil {
					return fmt.Errorf("failed to clean containerd configuration directory: %w", err)
				}
			}
			// On Windows, we'll just overwrite files
		}

		// Set proper permissions (Linux only)
		if platform.IsLinux() {
			if err := utils.RunSystemCommand("chmod", "-R", "0755", dir); err != nil {
				logrus.Warnf("Failed to set permissions for containerd directory %s: %v", dir, err)
			}
		}
	}
	return nil
}

func (i *Installer) installContainerd() error {
	// Check if we can skip installation
	if i.canSkipContainerdInstallation() {
		i.logger.Info("containerd is already installed and valid, skipping installation")
		return nil
	}

	// Clean up any corrupted installations before proceeding
	i.logger.Info("Cleaning up corrupted containerd installation files to start fresh")
	if err := i.cleanupExistingInstallation(); err != nil {
		i.logger.Warnf("Failed to cleanup existing containerd installation: %v", err)
		// Continue anyway - we'll install fresh
	}

	// Construct download URL
	fileName, downloadURL, err := i.constructContainerdDownloadURL()
	if err != nil {
		return fmt.Errorf("failed to construct containerd download URL: %w", err)
	}

	// Download the containerd tar file into temp directory
	fs := i.platform.FileSystem()
	paths := i.platform.Paths()
	tempFile := filepath.Join(paths.TempDir, fileName)

	// Clean up any existing temp files
	_ = fs.RemoveFile(tempFile)
	defer func() {
		_ = fs.RemoveFile(tempFile)
	}()

	i.logger.Infof("Downloading containerd from %s into %s", downloadURL, tempFile)
	if err := fs.DownloadFile(downloadURL, tempFile); err != nil {
		return fmt.Errorf("failed to download containerd from %s: %w", downloadURL, err)
	}

	// Extract containerd binaries
	if err := i.extractContainerd(tempFile); err != nil {
		return fmt.Errorf("failed to extract containerd binaries: %w", err)
	}

	// Set executable permissions (Linux only)
	if platform.IsLinux() {
		i.logger.Info("Setting executable permissions on containerd binaries")
		for _, binary := range containerdBinaries {
			binaryPath := filepath.Join(systemBinDir, binary)
			if err := utils.RunSystemCommand("chmod", "0755", binaryPath); err != nil {
				return fmt.Errorf("failed to set executable permissions on containerd binaries: %w", err)
			}
		}
	}

	return nil
}

func (i *Installer) extractContainerd(archivePath string) error {
	if platform.IsWindows() {
		// Windows: extract to Program Files\containerd
		i.logger.Infof("Extracting containerd binaries to %s", systemBinDir)
		return i.platform.FileSystem().ExtractTarGz(archivePath, i.platform.Paths().ContainerdConfigDir)
	}

	// Linux: extract to /usr/bin, stripping the 'bin/' prefix
	i.logger.Infof("Extracting containerd binaries to %s", systemBinDir)
	return utils.RunSystemCommand("tar", "-C", systemBinDir, "--strip-components=1", "-xzf", archivePath, "bin/")
}

func (i *Installer) canSkipContainerdInstallation() bool {
	fs := i.platform.FileSystem()

	// Check if containerd binary exists
	for _, binary := range containerdBinaries {
		binaryPath := filepath.Join(systemBinDir, binary)
		if !fs.FileExists(binaryPath) {
			i.logger.Debugf("containerd binary %s does not exist", binaryPath)
			return false
		}
	}

	// Verify containerd version is correct
	output, err := utils.RunCommandWithOutput(defaultContainerdBinaryDir, "--version")
	if err != nil {
		i.logger.Debugf("Failed to get containerd version from %s: %v", defaultContainerdBinaryDir, err)
		return false
	}
	versionMatch := strings.Contains(string(output), i.getContainerdVersion())
	if versionMatch {
		i.logger.Infof("containerd version %s is already installed", i.getContainerdVersion())
		return true
	}

	return false
}

// constructContainerdDownloadURL constructs the download URL for the specified containerd version
// it returns the file name and URL for downloading containerd
func (i *Installer) constructContainerdDownloadURL() (string, string, error) {
	containerdVersion := i.getContainerdVersion()
	arch, err := i.platform.FileSystem().GetArchitecture()
	if err != nil {
		return "", "", fmt.Errorf("failed to get architecture: %w", err)
	}

	var url, fileName string
	if platform.IsWindows() {
		// Windows uses a fixed architecture in the filename
		fileName = fmt.Sprintf("containerd-%s-windows-amd64.tar.gz", containerdVersion)
		url = fmt.Sprintf("https://github.com/containerd/containerd/releases/download/v%s/%s", containerdVersion, fileName)
	} else {
		fileName = fmt.Sprintf(containerdFileName, containerdVersion, arch)
		url = fmt.Sprintf(containerdDownloadURL, containerdVersion, containerdVersion, arch)
	}

	i.logger.Infof("Constructed containerd download URL: %s", url)
	return fileName, url, nil
}

// cleanupExistingInstallation removes any existing containerd installation that may be corrupted
func (i *Installer) cleanupExistingInstallation() error {
	i.logger.Debug("Cleaning up existing containerd installation files")
	fs := i.platform.FileSystem()

	// Try to stop containerd service
	svc := i.platform.Service()
	if svc.Exists("containerd") {
		_ = svc.Stop("containerd")
	}

	// On Linux, try to kill any running processes
	if platform.IsLinux() {
		if err := utils.RunSystemCommand("pkill", "-f", "containerd"); err != nil {
			i.logger.Debugf("No containerd processes found to kill (or pkill failed): %v", err)
		}
	}

	// Remove binaries
	for _, binary := range containerdBinaries {
		binaryPath := filepath.Join(systemBinDir, binary)
		if fs.FileExists(binaryPath) {
			i.logger.Debugf("Removing existing containerd binary: %s", binaryPath)
			if err := fs.RemoveFile(binaryPath); err != nil {
				i.logger.Warnf("Failed to remove %s: %v", binaryPath, err)
			}
		}
	}

	i.logger.Debug("Successfully cleaned up existing containerd installation")
	return nil
}

// configure configures containerd service and configuration files
func (i *Installer) configure() error {
	// Create containerd configuration
	if err := i.createContainerdConfigFile(); err != nil {
		return err
	}

	if platform.IsWindows() {
		return i.configureWindows()
	}
	return i.configureLinux()
}

func (i *Installer) configureLinux() error {
	// Create containerd systemd service
	if err := i.createLinuxServiceFile(); err != nil {
		return err
	}

	// Reload systemd to pick up the new containerd service configuration
	i.logger.Info("Reloading systemd to pick up containerd configuration changes")
	return i.platform.Service().ReloadDaemon()
}

func (i *Installer) configureWindows() error {
	// On Windows, containerd can register itself as a service
	i.logger.Info("Registering containerd as Windows service")

	containerdExe := filepath.Join(i.platform.Paths().ContainerdBinDir, "containerd.exe")
	configPath := filepath.Join(i.platform.Paths().ContainerdConfigDir, "config.toml")

	// Use containerd's built-in service registration
	_, err := utils.RunCommandWithOutput(containerdExe, "--register-service", "--config", configPath)
	if err != nil {
		// If built-in registration fails, use platform service manager
		i.logger.Warnf("Built-in service registration failed, using platform service manager: %v", err)

		svcConfig := &platform.ServiceConfig{
			Name:          "containerd",
			DisplayName:   "containerd container runtime",
			Description:   "containerd container runtime for Kubernetes",
			BinaryPath:    containerdExe,
			Args:          []string{"--config", configPath},
			RestartPolicy: platform.RestartAlways,
		}
		return i.platform.Service().Install(svcConfig)
	}

	return nil
}

// createLinuxServiceFile creates the containerd systemd service file
func (i *Installer) createLinuxServiceFile() error {
	containerdService := `[Unit]
Description=containerd container runtime
Documentation=https://containerd.io
After=network.target local-fs.target
[Service]
ExecStartPre=-/sbin/modprobe overlay
ExecStart=/usr/bin/containerd
Type=notify
Delegate=yes
KillMode=process
Restart=always
RestartSec=5
# Having non-zero Limit*s causes performance problems due to accounting overhead
# in the kernel. We recommend using cgroups to do container-local accounting.
LimitNPROC=infinity
LimitCORE=infinity
LimitNOFILE=infinity
# Comment TasksMax if your systemd version does not supports it.
# Only systemd 226 and above support this version.
TasksMax=infinity
OOMScoreAdjust=-999
[Install]
WantedBy=multi-user.target`

	// Create containerd service file using sudo-aware approach
	tempFile, err := utils.CreateTempFile("containerd-service-*.service", []byte(containerdService))
	if err != nil {
		return fmt.Errorf("failed to create temporary containerd service file: %w", err)
	}
	defer utils.CleanupTempFile(tempFile.Name())

	// Copy the temp file to the final location using sudo
	if err := utils.RunSystemCommand("cp", tempFile.Name(), containerdServiceFile); err != nil {
		return fmt.Errorf("failed to install containerd service file: %w", err)
	}

	// Set proper permissions
	if err := utils.RunSystemCommand("chmod", "644", containerdServiceFile); err != nil {
		return fmt.Errorf("failed to set containerd service file permissions: %w", err)
	}

	return nil
}

// createContainerdConfigFile creates the containerd configuration file
func (i *Installer) createContainerdConfigFile() error {
	var containerdConfig string

	if platform.IsWindows() {
		containerdConfig = i.generateWindowsConfig()
	} else {
		containerdConfig = i.generateLinuxConfig()
	}

	fs := i.platform.FileSystem()

	// Ensure config directory exists
	if err := fs.CreateDirectory(defaultContainerdConfigDir); err != nil {
		return fmt.Errorf("failed to create containerd config directory: %w", err)
	}

	// Write config file
	if err := fs.WriteFile(containerdConfigFile, []byte(containerdConfig), 0644); err != nil {
		return fmt.Errorf("failed to write containerd config file: %w", err)
	}

	return nil
}

func (i *Installer) generateLinuxConfig() string {
	paths := i.platform.Paths()
	return fmt.Sprintf(`version = 2
oom_score = 0
[plugins."io.containerd.grpc.v1.cri"]
	sandbox_image = "%s"
	[plugins."io.containerd.grpc.v1.cri".containerd]
		default_runtime_name = "runc"
		[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
			runtime_type = "io.containerd.runc.v2"
		[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
			BinaryName = "/usr/bin/runc"
			SystemdCgroup = true
		[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.untrusted]
			runtime_type = "io.containerd.runc.v2"
		[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.untrusted.options]
			BinaryName = "/usr/bin/runc"
	[plugins."io.containerd.grpc.v1.cri".cni]
		bin_dir = "%s"
		conf_dir = "%s"
	[plugins."io.containerd.grpc.v1.cri".registry]
		config_path = "/etc/containerd/certs.d"
	[plugins."io.containerd.grpc.v1.cri".registry.headers]
		X-Meta-Source-Client = ["azure/aks"]
[metrics]
	address = "%s"`,
		i.getPauseImage(),
		paths.CNIBinDir,
		paths.CNIConfDir,
		i.getMetricsAddress())
}

func (i *Installer) generateWindowsConfig() string {
	paths := i.platform.Paths()
	// Windows containerd config based on ECPWindowsHost reference
	return fmt.Sprintf(`version = 2
[plugins."io.containerd.grpc.v1.cri"]
	sandbox_image = "%s"
	[plugins."io.containerd.grpc.v1.cri".containerd]
		default_runtime_name = "runhcs-wcow-process"
		[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runhcs-wcow-process]
			runtime_type = "io.containerd.runhcs.v1"
	[plugins."io.containerd.grpc.v1.cri".cni]
		bin_dir = "%s"
		conf_dir = "%s"
[metrics]
	address = "%s"`,
		i.getWindowsPauseImage(),
		paths.CNIBinDir,
		paths.CNIConfDir,
		i.getMetricsAddress())
}

// Validate validates preconditions before execution
func (i *Installer) Validate(ctx context.Context) error {
	return nil
}

// GetName returns the step name
func (i *Installer) GetName() string {
	return "ContainerdInstaller"
}

// IsCompleted checks if containerd and required plugins are installed
func (i *Installer) IsCompleted(ctx context.Context) bool {
	fs := i.platform.FileSystem()

	// Check if containerd binaries are installed and functional
	if !i.canSkipContainerdInstallation() {
		return false
	}

	// Check if containerd config file exists
	if !fs.FileExists(containerdConfigFile) {
		return false
	}

	if platform.IsLinux() {
		// Check if containerd service file exists
		if !fs.FileExists(containerdServiceFile) {
			return false
		}

		// Verify systemd can parse the service file
		if err := utils.RunSystemCommand("systemctl", "check", "containerd"); err != nil {
			i.logger.Debugf("containerd service file is invalid: %v", err)
			return false
		}
	} else {
		// Windows: check if service is registered
		if !i.platform.Service().Exists("containerd") {
			return false
		}
	}

	return true
}

func (i *Installer) getContainerdVersion() string {
	if i.config.Containerd.Version != "" {
		return i.config.Containerd.Version
	}
	// Default to a known stable version if not specified
	return "1.7.20"
}

func (i *Installer) getPauseImage() string {
	if i.config.Containerd.PauseImage != "" {
		return i.config.Containerd.PauseImage
	}
	// Default pause image
	return "mcr.microsoft.com/oss/kubernetes/pause:3.6"
}

func (i *Installer) getWindowsPauseImage() string {
	if i.config.Containerd.PauseImage != "" {
		return i.config.Containerd.PauseImage
	}
	// Windows pause image (based on ECPWindowsHost)
	return "mcr.microsoft.com/oss/kubernetes/pause:3.10"
}

func (i *Installer) getMetricsAddress() string {
	if i.config.Containerd.MetricsAddress != "" {
		return i.config.Containerd.MetricsAddress
	}
	// Default metrics address
	return "0.0.0.0:10257"
}
