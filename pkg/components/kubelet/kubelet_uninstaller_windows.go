//go:build windows
// +build windows

package kubelet

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
	"go.goms.io/aks/AKSFlexNode/pkg/platform"
)

// UnInstaller handles kubelet cleanup operations on Windows
type UnInstaller struct {
	config   *config.Config
	logger   *logrus.Logger
	platform platform.Platform
}

// NewUnInstaller creates a new kubelet UnInstaller for Windows
func NewUnInstaller(logger *logrus.Logger) *UnInstaller {
	return &UnInstaller{
		config:   config.GetConfig(),
		logger:   logger,
		platform: platform.Current(),
	}
}

// GetName returns the step name
func (u *UnInstaller) GetName() string {
	return "KubeletUninstaller"
}

// Validate validates prerequisites for kubelet cleanup
func (u *UnInstaller) Validate(ctx context.Context) error {
	// No specific validation needed for cleanup
	return nil
}

// Execute removes kubelet configuration and service on Windows
func (u *UnInstaller) Execute(ctx context.Context) error {
	u.logger.Info("Cleaning up kubelet for Windows")

	// Step 1: Stop and remove kubelet service
	u.logger.Info("Step 1: Stopping and removing kubelet service")
	if err := u.removeKubeletService(); err != nil {
		u.logger.Warnf("Failed to remove kubelet service (continuing): %v", err)
	}

	// Step 2: Remove configuration files
	u.logger.Info("Step 2: Removing kubelet configuration files")
	if err := u.removeConfigFiles(); err != nil {
		u.logger.Warnf("Failed to remove config files (continuing): %v", err)
	}

	u.logger.Info("Kubelet cleanup completed")
	return nil
}

// IsCompleted checks if kubelet cleanup has been completed
func (u *UnInstaller) IsCompleted(ctx context.Context) bool {
	// Always return false to ensure cleanup is attempted
	return false
}

func (u *UnInstaller) removeKubeletService() error {
	// Stop the service first
	if err := u.platform.Service().Stop(kubeletServiceName); err != nil {
		u.logger.Debugf("Failed to stop kubelet service: %v", err)
	}

	// Uninstall the service
	if err := u.platform.Service().Uninstall(kubeletServiceName); err != nil {
		return err
	}

	u.logger.Info("Kubelet service removed successfully")
	return nil
}

func (u *UnInstaller) removeConfigFiles() error {
	// Files to remove
	filesToRemove := []string{
		kubeletKubeconfigPath,
		kubeletTokenScriptPath,
		kubeletConfigPath,
	}

	for _, file := range filesToRemove {
		if _, err := os.Stat(file); err == nil {
			if err := os.Remove(file); err != nil {
				u.logger.Warnf("Failed to remove %s: %v", file, err)
			} else {
				u.logger.Debugf("Removed: %s", file)
			}
		}
	}

	return nil
}
