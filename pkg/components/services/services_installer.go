package services

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
	"go.goms.io/aks/AKSFlexNode/pkg/platform"
)

// Installer handles enabling and starting system services
type Installer struct {
	config   *config.Config
	logger   *logrus.Logger
	platform platform.Platform
}

// NewInstaller creates a new services Installer
func NewInstaller(logger *logrus.Logger) *Installer {
	return &Installer{
		config:   config.GetConfig(),
		logger:   logger,
		platform: platform.Current(),
	}
}

// Execute enables and starts required services (containerd and kubelet)
func (i *Installer) Execute(ctx context.Context) error {
	i.logger.Info("Enabling and starting services")

	svc := i.platform.Service()

	// Reload service daemon (systemd on Linux, no-op on Windows)
	if err := svc.ReloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload service daemon: %w", err)
	}

	// Enable and start containerd
	i.logger.Info("Enabling and starting containerd service")
	if err := i.enableAndStartService(ContainerdService); err != nil {
		i.logger.Errorf("Failed to enable and start containerd: %v", err)
		return fmt.Errorf("failed to enable and start containerd: %w", err)
	}

	// Restart containerd to pick up CNI configuration changes
	i.logger.Info("Restarting containerd service to apply CNI configuration")
	if err := svc.Restart(ContainerdService); err != nil {
		i.logger.Errorf("Failed to restart containerd: %v", err)
		return fmt.Errorf("failed to restart containerd for CNI reload: %w", err)
	}

	// Enable and start kubelet
	i.logger.Info("Enabling and starting kubelet service")
	if err := i.enableAndStartService(KubeletService); err != nil {
		i.logger.Errorf("Failed to enable and start kubelet: %v", err)
		return fmt.Errorf("failed to enable and start kubelet: %w", err)
	}

	// Wait for kubelet to start and validate it's running properly
	i.logger.Info("Waiting for kubelet to start...")
	if err := svc.WaitForService(KubeletService, int(ServiceStartupTimeout/time.Second)); err != nil {
		return fmt.Errorf("kubelet failed to start properly: %w", err)
	}

	// Enable and start node-problem-detector (if available)
	i.logger.Info("Enabling and starting node-problem-detector service")
	if err := i.enableAndStartService(NPDService); err != nil {
		i.logger.Warnf("Failed to enable and start node-problem-detector: %v (continuing anyway)", err)
		// NPD is optional, don't fail the bootstrap
	}

	i.logger.Info("All services enabled and started successfully")
	return nil
}

// enableAndStartService enables and starts a service
func (i *Installer) enableAndStartService(name string) error {
	svc := i.platform.Service()

	// Enable the service to start on boot
	if err := svc.Enable(name); err != nil {
		return fmt.Errorf("failed to enable service %s: %w", name, err)
	}

	// Start the service
	if err := svc.Start(name); err != nil {
		return fmt.Errorf("failed to start service %s: %w", name, err)
	}

	return nil
}

// IsCompleted checks if containerd and kubelet services are enabled and running
func (i *Installer) IsCompleted(ctx context.Context) bool {
	// always return false to ensure services are reenabled each time
	return false
}

// Validate validates prerequisites for enabling services
func (i *Installer) Validate(ctx context.Context) error {
	i.logger.Debug("Validating prerequisites for enabling services")
	// No specific prerequisites for enabling services
	return nil
}

// GetName returns the step name
func (i *Installer) GetName() string {
	return "ServicesEnabled"
}
