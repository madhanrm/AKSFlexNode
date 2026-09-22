package kube_binaries

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

// Installer handles Kube binaries installation operations
type Installer struct {
	config   *config.Config
	logger   *logrus.Logger
	platform platform.Platform
}

// NewInstaller creates a new Kube binaries Installer
func NewInstaller(logger *logrus.Logger) *Installer {
	return &Installer{
		config:   config.GetConfig(),
		logger:   logger,
		platform: platform.Current(),
	}
}

// Execute downloads and installs Kube binaries (kubelet, kubectl, kubeadm)
func (i *Installer) Execute(ctx context.Context) error {
	i.logger.Infof("Installing Kube Binaries of version %s", i.config.GetKubernetesVersion())

	// Download and install Kubernetes binaries
	if err := i.installKubeBinaries(); err != nil {
		return fmt.Errorf("failed to install Kubernetes: %w", err)
	}

	i.logger.Info("Kubernetes binaries installed successfully")
	return nil
}

func (i *Installer) installKubeBinaries() error {
	// Clean up any corrupted installations before proceeding
	i.logger.Info("Cleaning up corrupted Kubernetes installation files to start fresh")
	if err := i.cleanupExistingInstallation(); err != nil {
		i.logger.Warnf("Failed to cleanup existing Kubernetes installation: %v", err)
		// Continue anyway - we'll install fresh
	}

	// Construct download URL
	fileName, url, err := i.constructKubeBinariesDownloadURL()
	if err != nil {
		return fmt.Errorf("failed to construct Kubernetes download URL: %w", err)
	}

	// Download the Kubernetes tar file into temp directory
	fs := i.platform.FileSystem()
	paths := i.platform.Paths()
	tempFile := filepath.Join(paths.TempDir, fileName)

	// Clean up any existing temp files
	_ = fs.RemoveFile(tempFile)
	defer func() {
		_ = fs.RemoveFile(tempFile)
	}()

	// Download Kube binaries with validation
	i.logger.Infof("Downloading Kube binaries from %s into %s", url, tempFile)
	if err := fs.DownloadFile(url, tempFile); err != nil {
		return fmt.Errorf("failed to download Kube binaries from %s: %w", url, err)
	}

	// Ensure bin directory exists
	if err := fs.CreateDirectory(binDir); err != nil {
		return fmt.Errorf("failed to create bin directory %s: %w", binDir, err)
	}

	// Extract Kubernetes binaries
	i.logger.Infof("Extracting Kubernetes binaries to %s", binDir)
	if err := i.extractKubeBinaries(tempFile); err != nil {
		return fmt.Errorf("failed to extract Kubernetes binaries: %w", err)
	}

	// Ensure all extracted binaries are executable and have proper permissions (Linux only)
	if platform.IsLinux() {
		i.logger.Info("Setting executable permissions on Kubernetes binaries")
		for _, binaryPath := range kubeBinariesPaths {
			if err := utils.RunSystemCommand("chmod", "0755", binaryPath); err != nil {
				return fmt.Errorf("failed to set executable permissions on Kubernetes binaries: %w", err)
			}
		}
	}

	return nil
}

func (i *Installer) extractKubeBinaries(archivePath string) error {
	if platform.IsWindows() {
		// Windows: extract to temp location first, then move binaries to C:\k
		fs := i.platform.FileSystem()
		tempExtractDir := filepath.Join(i.platform.Paths().TempDir, "k8s-extract")

		// Clean up temp directory if it exists
		_ = fs.RemoveDirectory(tempExtractDir)

		// Create temp extraction directory
		if err := fs.CreateDirectory(tempExtractDir); err != nil {
			return fmt.Errorf("failed to create temp extraction directory: %w", err)
		}
		defer func() {
			_ = fs.RemoveDirectory(tempExtractDir)
		}()

		// Extract to temp directory
		if err := fs.ExtractTarGz(archivePath, tempExtractDir); err != nil {
			return fmt.Errorf("failed to extract archive: %w", err)
		}

		// Move required binaries from nested path to C:\k
		srcDir := filepath.Join(tempExtractDir, "kubernetes", "node", "bin")
		binaries := []string{"kubelet.exe", "kubectl.exe", "kubeadm.exe", "kube-proxy.exe"}

		for _, bin := range binaries {
			srcPath := filepath.Join(srcDir, bin)
			dstPath := filepath.Join(binDir, bin)

			// Check if source exists
			if !fs.FileExists(srcPath) {
				i.logger.Debugf("Binary %s not found in archive, skipping", bin)
				continue
			}

			// Read source file
			i.logger.Debugf("Copying %s to %s", srcPath, dstPath)
			content, err := fs.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", srcPath, err)
			}

			// Write to destination
			if err := fs.WriteFile(dstPath, content, 0755); err != nil {
				return fmt.Errorf("failed to write %s: %w", dstPath, err)
			}
		}

		return nil
	}

	// Linux: extract to /usr/local/bin, stripping the 'kubernetes/node/bin/' prefix
	return utils.RunSystemCommand("tar", "-C", binDir, "--strip-components=3", "-xzf", archivePath, kubernetesTarPath)
}

// IsCompleted checks if all Kube binaries are installed
func (i *Installer) IsCompleted(ctx context.Context) bool {
	if i.canSkipKubernetesInstallation() {
		i.logger.Info("Kube binaries are already installed and valid, skipping installation")
		return true
	}
	return false
}

// Validate validates prerequisites for Kube binaries installation
func (i *Installer) Validate(ctx context.Context) error {
	// Verify network connectivity for download (basic check)
	kubernetesVersion := i.config.GetKubernetesVersion()
	if kubernetesVersion == "" {
		return fmt.Errorf("kubernetes version not specified")
	}
	return nil
}

// canSkipKubernetesInstallation checks if all Kube binaries are installed with the correct version
func (i *Installer) canSkipKubernetesInstallation() bool {
	fs := i.platform.FileSystem()
	for _, binaryPath := range kubeBinariesPaths {
		if !fs.FileExists(binaryPath) {
			i.logger.Debugf("Kubernetes binary not found: %s", binaryPath)
			return false
		}

		// Check version for kubelet (main component)
		if binaryPath == kubeletPath {
			if !i.isKubeletVersionCorrect() {
				i.logger.Debugf("Kubelet version is incorrect")
				return false
			}
		}
	}
	return true
}

// isKubeletVersionCorrect checks if the installed kubelet version matches the expected version
func (i *Installer) isKubeletVersionCorrect() bool {
	output, err := utils.RunCommandWithOutput(kubeletPath, "--version")
	if err != nil {
		i.logger.Debugf("Failed to get kubelet version: %v", err)
		return false
	}

	// Check if version output contains expected version
	return strings.Contains(string(output), i.config.GetKubernetesVersion())
}

// cleanupExistingInstallation removes any existing Kubernetes installation that may be corrupted
func (i *Installer) cleanupExistingInstallation() error {
	i.logger.Debug("Cleaning up existing Kubernetes installation files")
	fs := i.platform.FileSystem()

	// Try to stop kubelet daemon process (best effort) - only on Linux
	if platform.IsLinux() {
		if err := utils.RunSystemCommand("pkill", "-f", "kubelet"); err != nil {
			i.logger.Debugf("No kubelet processes found to kill (or pkill failed): %v", err)
		}
	}

	// List of binaries to clean up
	for _, binaryPath := range kubeBinariesPaths {
		if fs.FileExists(binaryPath) {
			i.logger.Debugf("Removing existing Kubernetes binary: %s", binaryPath)
			if err := fs.RemoveFile(binaryPath); err != nil {
				i.logger.Warnf("Failed to remove %s: %v", binaryPath, err)
			}
		}
	}

	i.logger.Debug("Successfully cleaned up stale Kubernetes installation")
	return nil
}

// constructKubeBinariesDownloadURL constructs the download URL for the specified Kubernetes version
// it returns the file name and URL for downloading Kube binaries
func (i *Installer) constructKubeBinariesDownloadURL() (string, string, error) {
	arch, err := i.platform.FileSystem().GetArchitecture()
	if err != nil {
		return "", "", fmt.Errorf("failed to get architecture: %w", err)
	}

	kubernetesVersion := i.config.GetKubernetesVersion()
	urlTemplate := i.getKubernetesURLTemplate()

	var url, fileName string
	if platform.IsWindows() {
		// Windows always uses amd64
		fileName = kubernetesFileName
		url = fmt.Sprintf(urlTemplate, kubernetesVersion, "amd64")
	} else {
		fileName = fmt.Sprintf(kubernetesFileName, arch)
		url = fmt.Sprintf(urlTemplate, kubernetesVersion, arch)
	}

	i.logger.Infof("Constructed Kubernetes download URL: %s", url)
	return fileName, url, nil
}

func (i *Installer) getKubernetesURLTemplate() string {
	if i.config.Kubernetes.URLTemplate != "" {
		return i.config.Kubernetes.URLTemplate
	}
	// Default URL template for Kubernetes binaries
	return defaultKubernetesURLTemplate
}

// GetName returns the step name
func (i *Installer) GetName() string {
	return "KubeBinariesInstaller"
}
