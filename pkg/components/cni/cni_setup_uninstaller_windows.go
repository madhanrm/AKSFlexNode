//go:build windows
// +build windows

package cni

import (
	"context"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
	"go.goms.io/aks/AKSFlexNode/pkg/platform"
)

// UnInstaller handles Calico CNI cleanup operations on Windows
type UnInstaller struct {
	config   *config.Config
	logger   *logrus.Logger
	platform platform.Platform
}

// NewUnInstaller creates a new CNI UnInstaller for Windows
func NewUnInstaller(logger *logrus.Logger) *UnInstaller {
	return &UnInstaller{
		config:   config.GetConfig(),
		logger:   logger,
		platform: platform.Current(),
	}
}

// GetName returns the step name
func (u *UnInstaller) GetName() string {
	return "CNICleanup"
}

// Validate validates prerequisites for CNI cleanup
func (u *UnInstaller) Validate(ctx context.Context) error {
	// No special validation needed for cleanup
	return nil
}

// Execute removes Calico CNI configuration and binaries
func (u *UnInstaller) Execute(ctx context.Context) error {
	u.logger.Info("Cleaning up Calico CNI for Windows")

	// Step 1: Remove CNI configuration
	u.logger.Info("Step 1: Removing CNI configuration")
	if err := u.removeCNIConfig(); err != nil {
		u.logger.Warnf("Failed to remove CNI config (continuing): %v", err)
	}

	// Step 2: Remove CNI binaries
	u.logger.Info("Step 2: Removing CNI binaries")
	if err := u.removeCNIBinaries(); err != nil {
		u.logger.Warnf("Failed to remove CNI binaries (continuing): %v", err)
	}

	// Step 3: Remove Calico Windows directory
	u.logger.Info("Step 3: Removing Calico Windows directory")
	if err := u.removeCalicoDir(); err != nil {
		u.logger.Warnf("Failed to remove Calico directory (continuing): %v", err)
	}

	u.logger.Info("Calico CNI cleanup completed")
	return nil
}

// IsCompleted checks if CNI cleanup has been completed
func (u *UnInstaller) IsCompleted(ctx context.Context) bool {
	// Always return false to ensure cleanup is attempted
	return false
}

func (u *UnInstaller) removeCNIConfig() error {
	// Remove CNI configuration file
	configPath := filepath.Join(DefaultCNIConfDir, calicoConfigFile)
	if _, err := os.Stat(configPath); err == nil {
		if err := os.Remove(configPath); err != nil {
			return err
		}
		u.logger.Infof("Removed CNI config: %s", configPath)
	}

	// Remove config.ps1
	configPS1 := filepath.Join(CalicoDir, "config.ps1")
	if _, err := os.Stat(configPS1); err == nil {
		if err := os.Remove(configPS1); err != nil {
			u.logger.Warnf("Failed to remove config.ps1: %v", err)
		}
	}

	return nil
}

func (u *UnInstaller) removeCNIBinaries() error {
	// Remove CNI plugin binaries
	for _, plugin := range requiredCNIPlugins {
		pluginPath := filepath.Join(DefaultCNIBinDir, plugin)
		if _, err := os.Stat(pluginPath); err == nil {
			if err := os.Remove(pluginPath); err != nil {
				u.logger.Warnf("Failed to remove plugin %s: %v", plugin, err)
			} else {
				u.logger.Infof("Removed CNI plugin: %s", pluginPath)
			}
		}
	}

	// Also remove optional plugins
	optionalPlugins := []string{hostLocalPlugin, winBridgePlugin, winOverlayPlugin, flannelPlugin}
	for _, plugin := range optionalPlugins {
		pluginPath := filepath.Join(DefaultCNIBinDir, plugin)
		if _, err := os.Stat(pluginPath); err == nil {
			os.Remove(pluginPath)
		}
	}

	return nil
}

func (u *UnInstaller) removeCalicoDir() error {
	// Remove Calico Windows directory
	if _, err := os.Stat(CalicoDir); err == nil {
		if err := os.RemoveAll(CalicoDir); err != nil {
			return err
		}
		u.logger.Infof("Removed Calico directory: %s", CalicoDir)
	}

	// Remove Calico data directory
	if _, err := os.Stat(CalicoDataDir); err == nil {
		if err := os.RemoveAll(CalicoDataDir); err != nil {
			u.logger.Warnf("Failed to remove Calico data directory: %v", err)
		}
	}

	// Remove Calico log directory
	if _, err := os.Stat(CalicoLogDir); err == nil {
		if err := os.RemoveAll(CalicoLogDir); err != nil {
			u.logger.Warnf("Failed to remove Calico log directory: %v", err)
		}
	}

	// Remove Calico etc directory
	if _, err := os.Stat(CalicoEtcDir); err == nil {
		if err := os.RemoveAll(CalicoEtcDir); err != nil {
			u.logger.Warnf("Failed to remove Calico etc directory: %v", err)
		}
	}

	return nil
}
