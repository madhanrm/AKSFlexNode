//go:build windows
// +build windows

package runhcs

import (
	"context"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
	"go.goms.io/aks/AKSFlexNode/pkg/platform"
)

// UnInstaller handles runhcs removal on Windows
// Note: runhcs is bundled with containerd, so uninstallation is typically handled
// by the containerd uninstaller
type UnInstaller struct {
	config   *config.Config
	logger   *logrus.Logger
	platform platform.Platform
}

// NewUnInstaller creates a new runhcs unInstaller
func NewUnInstaller(logger *logrus.Logger) *UnInstaller {
	return &UnInstaller{
		config:   config.GetConfig(),
		logger:   logger,
		platform: platform.Current(),
	}
}

// GetName returns the cleanup step name
func (u *UnInstaller) GetName() string {
	return "Runhcs_Uninstaller"
}

// Execute removes runhcs shim
// Note: This is typically a no-op since runhcs is bundled with containerd
func (u *UnInstaller) Execute(ctx context.Context) error {
	u.logger.Info("Uninstalling runhcs (bundled with containerd)")

	shimPath := filepath.Join(u.platform.Paths().ContainerdBinDir, hcsshimFileName)
	fs := u.platform.FileSystem()

	if fs.FileExists(shimPath) {
		if err := fs.RemoveFile(shimPath); err != nil {
			u.logger.Debugf("Failed to remove runhcs shim at %s: %v", shimPath, err)
			// Not a critical error - containerd uninstaller will clean up the directory
		}
	}

	u.logger.Info("Runhcs uninstalled successfully")
	return nil
}

// IsCompleted checks if runhcs has been removed
func (u *UnInstaller) IsCompleted(ctx context.Context) bool {
	shimPath := filepath.Join(u.platform.Paths().ContainerdBinDir, hcsshimFileName)
	return !u.platform.FileSystem().FileExists(shimPath)
}
