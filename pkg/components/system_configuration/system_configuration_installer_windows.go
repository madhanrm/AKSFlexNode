//go:build windows
// +build windows

package system_configuration

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
	"go.goms.io/aks/AKSFlexNode/pkg/platform"
)

// Installer handles system configuration for Windows
type Installer struct {
	config   *config.Config
	logger   *logrus.Logger
	platform platform.Platform
}

// NewInstaller creates a new system configuration Installer
func NewInstaller(logger *logrus.Logger) *Installer {
	return &Installer{
		config:   config.GetConfig(),
		logger:   logger,
		platform: platform.Current(),
	}
}

// Execute configures Windows system settings for Kubernetes
func (i *Installer) Execute(ctx context.Context) error {
	i.logger.Info("Configuring Windows system settings")

	// Enable IP forwarding for container networking
	if err := i.enableIPForwarding(); err != nil {
		i.logger.Warnf("Failed to enable IP forwarding: %v", err)
		// Continue - not all Windows versions require this
	}

	// Configure Windows Firewall rules for Kubernetes
	if err := i.configureFirewall(); err != nil {
		i.logger.Warnf("Failed to configure firewall: %v", err)
		// Continue - firewall rules can be configured manually
	}

	// Create required directories
	if err := i.createRequiredDirectories(); err != nil {
		return err
	}

	i.logger.Info("Windows system configuration completed")
	return nil
}

// enableIPForwarding enables IP forwarding on Windows
func (i *Installer) enableIPForwarding() error {
	i.logger.Debug("Enabling IP forwarding")
	cmd := i.platform.Command()

	// Enable IP forwarding using PowerShell
	_, err := cmd.RunWithOutput(context.Background(), "powershell", "-Command",
		"Set-NetIPInterface -Forwarding Enabled -PolicyStore ActiveStore")
	if err != nil {
		return err
	}

	i.logger.Info("IP forwarding enabled")
	return nil
}

// configureFirewall configures Windows Firewall for Kubernetes
func (i *Installer) configureFirewall() error {
	i.logger.Debug("Configuring Windows Firewall for Kubernetes")
	cmd := i.platform.Command()

	// Allow kubelet port (10250)
	_, _ = cmd.RunWithOutput(context.Background(), "netsh", "advfirewall", "firewall", "add", "rule",
		"name=kubelet", "dir=in", "action=allow", "protocol=tcp", "localport=10250")

	// Allow kubelet healthz port (10248)
	_, _ = cmd.RunWithOutput(context.Background(), "netsh", "advfirewall", "firewall", "add", "rule",
		"name=kubelet-healthz", "dir=in", "action=allow", "protocol=tcp", "localport=10248")

	// Allow kubelet read-only port if enabled (10255)
	_, _ = cmd.RunWithOutput(context.Background(), "netsh", "advfirewall", "firewall", "add", "rule",
		"name=kubelet-readonly", "dir=in", "action=allow", "protocol=tcp", "localport=10255")

	i.logger.Info("Windows Firewall configured for Kubernetes")
	return nil
}

// createRequiredDirectories creates directories needed for Windows Kubernetes
func (i *Installer) createRequiredDirectories() error {
	i.logger.Debug("Creating required directories")
	fs := i.platform.FileSystem()
	paths := i.platform.Paths()

	dirs := []string{
		paths.KubeletConfigDir,
		paths.KubeletDataDir,
		paths.KubeletManifests,
		paths.CNIBinDir,
		paths.CNIConfDir,
	}

	for _, dir := range dirs {
		if err := fs.CreateDirectory(dir); err != nil {
			return err
		}
	}

	i.logger.Info("Required directories created")
	return nil
}

// IsCompleted checks if system configuration has been applied
func (i *Installer) IsCompleted(ctx context.Context) bool {
	// Check if required directories exist
	fs := i.platform.FileSystem()
	paths := i.platform.Paths()

	return fs.DirectoryExists(paths.KubeletConfigDir) &&
		fs.DirectoryExists(paths.KubeletDataDir)
}

// Validate validates the system configuration installation
func (i *Installer) Validate(ctx context.Context) error {
	return nil
}

// GetName returns the step name
func (i *Installer) GetName() string {
	return "SystemConfigured"
}
