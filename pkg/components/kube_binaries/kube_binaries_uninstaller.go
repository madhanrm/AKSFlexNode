package kube_binaries

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/platform"
)

// UnInstaller handles Kubernetes components removal operations
type UnInstaller struct {
	logger   *logrus.Logger
	platform platform.Platform
}

// NewUnInstaller creates a new Kubernetes components unInstaller
func NewUnInstaller(logger *logrus.Logger) *UnInstaller {
	return &UnInstaller{
		logger:   logger,
		platform: platform.Current(),
	}
}

// GetName returns the cleanup step name
func (u *UnInstaller) GetName() string {
	return "KubernetesComponentsRemoved"
}

// Execute removes Kubernetes components
func (u *UnInstaller) Execute(ctx context.Context) error {
	u.logger.Info("Removing Kubernetes binaries")
	fs := u.platform.FileSystem()

	// Remove Kubernetes binaries
	for _, binaryPath := range kubeBinariesPaths {
		if fs.FileExists(binaryPath) {
			u.logger.Debugf("Removing Kubernetes binary: %s", binaryPath)
			if err := fs.RemoveFile(binaryPath); err != nil {
				u.logger.Warnf("Failed to remove %s: %v", binaryPath, err)
			}
		}
	}

	u.logger.Info("Kubernetes binaries removal completed")
	return nil
}

// IsCompleted checks if Kubernetes components have been removed
func (u *UnInstaller) IsCompleted(ctx context.Context) bool {
	fs := u.platform.FileSystem()
	return !fs.FileExists(kubeletPath)
}
