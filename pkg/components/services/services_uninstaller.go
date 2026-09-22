package services

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
	"go.goms.io/aks/AKSFlexNode/pkg/platform"
)

// UnInstaller handles stopping and disabling system services
type UnInstaller struct {
	config   *config.Config
	logger   *logrus.Logger
	platform platform.Platform
}

// NewUnInstaller creates a new services unInstaller
func NewUnInstaller(logger *logrus.Logger) *UnInstaller {
	return &UnInstaller{
		config:   config.GetConfig(),
		logger:   logger,
		platform: platform.Current(),
	}
}

// GetName returns the cleanup step name
func (su *UnInstaller) GetName() string {
	return "ServicesDisabled"
}

// Execute stops and disables services
func (su *UnInstaller) Execute(ctx context.Context) error {
	su.logger.Info("Stopping and disabling services")

	svc := su.platform.Service()

	// Stop and disable node-problem-detector (optional)
	if svc.Exists(NPDService) {
		su.logger.Info("Stopping and disabling node-problem-detector service")
		if err := svc.Stop(NPDService); err != nil {
			su.logger.Warnf("Failed to stop node-problem-detector: %v", err)
		}
		if err := svc.Disable(NPDService); err != nil {
			su.logger.Warnf("Failed to disable node-problem-detector: %v", err)
		}
	}

	// Stop and disable kubelet
	if svc.Exists(KubeletService) {
		su.logger.Info("Stopping and disabling kubelet service")
		if err := svc.Stop(KubeletService); err != nil {
			su.logger.Warnf("Failed to stop kubelet: %v", err)
		}
		if err := svc.Disable(KubeletService); err != nil {
			su.logger.Warnf("Failed to disable kubelet: %v", err)
		}
	}

	// Stop and disable containerd
	if svc.Exists(ContainerdService) {
		su.logger.Info("Stopping and disabling containerd service")
		if err := svc.Stop(ContainerdService); err != nil {
			su.logger.Warnf("Failed to stop containerd: %v", err)
		}
		if err := svc.Disable(ContainerdService); err != nil {
			su.logger.Warnf("Failed to disable containerd: %v", err)
		}
	}

	su.logger.Info("Services stopped and disabled successfully")
	return nil
}

// IsCompleted checks if services have been stopped and disabled
func (su *UnInstaller) IsCompleted(ctx context.Context) bool {
	svc := su.platform.Service()
	// Services are considered completed if they are not active
	return !svc.IsActive(ContainerdService) && !svc.IsActive(KubeletService)
}
