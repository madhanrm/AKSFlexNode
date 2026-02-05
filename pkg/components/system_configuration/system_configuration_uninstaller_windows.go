//go:build windows
// +build windows

package system_configuration

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
	"go.goms.io/aks/AKSFlexNode/pkg/platform"
)

// UnInstaller handles system configuration cleanup on Windows
type UnInstaller struct {
	config   *config.Config
	logger   *logrus.Logger
	platform platform.Platform
}

// NewUnInstaller creates a new system configuration unInstaller
func NewUnInstaller(logger *logrus.Logger) *UnInstaller {
	return &UnInstaller{
		config:   config.GetConfig(),
		logger:   logger,
		platform: platform.Current(),
	}
}

// GetName returns the cleanup step name
func (su *UnInstaller) GetName() string {
	return "SystemCleanup"
}

// Execute removes system configuration and resets settings
func (su *UnInstaller) Execute(ctx context.Context) error {
	su.logger.Info("Cleaning up Windows system configuration")

	// Remove firewall rules
	if err := su.cleanupFirewallRules(); err != nil {
		su.logger.Warnf("Failed to cleanup firewall rules: %v", err)
	}

	su.logger.Info("Windows system configuration cleanup completed")
	return nil
}

// cleanupFirewallRules removes Kubernetes-related firewall rules
func (su *UnInstaller) cleanupFirewallRules() error {
	cmd := su.platform.Command()

	// Remove kubelet firewall rules
	_, _ = cmd.RunWithOutput(context.Background(), "netsh", "advfirewall", "firewall", "delete", "rule",
		"name=kubelet")
	_, _ = cmd.RunWithOutput(context.Background(), "netsh", "advfirewall", "firewall", "delete", "rule",
		"name=kubelet-healthz")
	_, _ = cmd.RunWithOutput(context.Background(), "netsh", "advfirewall", "firewall", "delete", "rule",
		"name=kubelet-readonly")

	su.logger.Info("Firewall rules removed")
	return nil
}

// IsCompleted checks if system configuration has been removed
func (su *UnInstaller) IsCompleted(ctx context.Context) bool {
	// Windows cleanup is considered complete after execution
	return true
}
