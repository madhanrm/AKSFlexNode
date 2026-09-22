//go:build windows
// +build windows

package runhcs

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
	"go.goms.io/aks/AKSFlexNode/pkg/platform"
)

// Installer handles runhcs container runtime shim installation on Windows
// Note: runhcs is typically bundled with containerd on Windows, so this installer
// mainly verifies the installation rather than downloading separately
type Installer struct {
	config   *config.Config
	logger   *logrus.Logger
	platform platform.Platform
}

// NewInstaller creates a new runhcs Installer
func NewInstaller(logger *logrus.Logger) *Installer {
	return &Installer{
		config:   config.GetConfig(),
		logger:   logger,
		platform: platform.Current(),
	}
}

// GetName returns the step name
func (i *Installer) GetName() string {
	return "Runhcs_Installer"
}

// Execute verifies the runhcs container runtime shim is installed
// On Windows, runhcs is bundled with containerd, so this mainly validates the installation
func (i *Installer) Execute(ctx context.Context) error {
	i.logger.Info("Verifying runhcs (Windows container runtime shim)")

	// runhcs should be installed as part of containerd on Windows
	// The shim binary is typically at: C:\Program Files\containerd\bin\containerd-shim-runhcs-v1.exe
	shimPath := filepath.Join(i.platform.Paths().ContainerdBinDir, hcsshimFileName)

	fs := i.platform.FileSystem()
	if !fs.FileExists(shimPath) {
		return fmt.Errorf("runhcs shim not found at %s - ensure containerd is properly installed", shimPath)
	}

	i.logger.Infof("runhcs shim verified at %s", shimPath)
	return nil
}

// IsCompleted checks if runhcs is installed
func (i *Installer) IsCompleted(ctx context.Context) bool {
	shimPath := filepath.Join(i.platform.Paths().ContainerdBinDir, hcsshimFileName)
	return i.platform.FileSystem().FileExists(shimPath)
}

// Validate validates prerequisites before verifying runhcs
func (i *Installer) Validate(ctx context.Context) error {
	// Check that containerd directory exists
	if !i.platform.FileSystem().DirectoryExists(i.platform.Paths().ContainerdBinDir) {
		return fmt.Errorf("containerd bin directory does not exist at %s - install containerd first", i.platform.Paths().ContainerdBinDir)
	}
	return nil
}

// isRunhcsVersionCorrect checks if the installed runhcs/shim version matches expected
func (i *Installer) isRunhcsVersionCorrect() bool {
	shimPath := filepath.Join(i.platform.Paths().ContainerdBinDir, hcsshimFileName)

	// containerd-shim-runhcs-v1 --version output contains version info
	cmd := i.platform.Command()
	output, err := cmd.RunWithOutput(context.Background(), shimPath, "--version")
	if err != nil {
		i.logger.Debugf("Failed to get runhcs version: %v", err)
		return false
	}

	// Version is embedded in containerd, just verify it runs
	return strings.Contains(output, "containerd") || strings.Contains(output, "runhcs")
}
