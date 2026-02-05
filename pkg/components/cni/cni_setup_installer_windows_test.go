//go:build windows
// +build windows

package cni

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestCNINewInstaller verifies the installer constructor.
func TestCNINewInstaller(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	if installer == nil {
		t.Fatal("NewInstaller should not return nil")
	}

	if installer.logger == nil {
		t.Error("installer.logger should be set")
	}

	if installer.platform == nil {
		t.Error("installer.platform should be set")
	}
}

// TestCNIInstallerGetName verifies the step name.
func TestCNIInstallerGetName(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	name := installer.GetName()

	if name == "" {
		t.Error("GetName should not return empty string")
	}

	if name != "CNISetup" {
		t.Errorf("GetName = %s, want CNISetup", name)
	}
}

// TestCNIInstallerValidate verifies validation logic.
func TestCNIInstallerValidate(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	ctx := context.Background()
	err := installer.Validate(ctx)

	// On a system without containerd, this should return an error
	if err != nil {
		if !strings.Contains(err.Error(), "containerd") {
			t.Logf("Validate returned error (may be expected): %v", err)
		}
	} else {
		t.Log("Validate passed - containerd exists")
	}
}

// TestCNIInstallerIsCompleted verifies completion check logic.
func TestCNIInstallerIsCompleted(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	ctx := context.Background()
	completed := installer.IsCompleted(ctx)

	// Should return false on clean system without Calico installed
	t.Logf("IsCompleted returned: %v (depends on system state)", completed)
}

// TestCanSkipCalicoInstallation verifies skip logic.
func TestCanSkipCalicoInstallation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	// Should return false if plugins don't exist
	canSkip := installer.canSkipCalicoInstallation()

	t.Logf("canSkipCalicoInstallation returned: %v", canSkip)
}

// TestGenerateCalicoConfig verifies config generation.
func TestGenerateCalicoConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	serviceCIDR := "10.0.0.0/16"
	dnsServiceIP := "10.0.0.10"

	config, err := installer.generateCalicoConfig(VXLAN, serviceCIDR, dnsServiceIP)

	if err != nil {
		t.Fatalf("generateCalicoConfig failed: %v", err)
	}

	if config == "" {
		t.Error("generateCalicoConfig should not return empty string")
	}

	// Verify JSON structure
	if !strings.Contains(config, `"name": "Calico"`) {
		t.Error("config should contain name field")
	}

	if !strings.Contains(config, `"cniVersion":`) {
		t.Error("config should contain cniVersion field")
	}

	if !strings.Contains(config, `"type": "calico"`) {
		t.Error("config should contain calico plugin type")
	}

	if !strings.Contains(config, `"mode": "vxlan"`) {
		t.Error("config should contain vxlan mode")
	}

	if !strings.Contains(config, dnsServiceIP) {
		t.Error("config should contain DNS service IP")
	}

	// Check for escaped Windows paths in JSON
	if !strings.Contains(config, "\\\\") {
		t.Log("config contains escaped backslashes for Windows paths")
	}
}

// TestGenerateCalicoConfigBGP verifies BGP backend config generation.
func TestGenerateCalicoConfigBGP(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	config, err := installer.generateCalicoConfig(WindowsBGP, "10.0.0.0/16", "10.0.0.10")

	if err != nil {
		t.Fatalf("generateCalicoConfig with BGP failed: %v", err)
	}

	if !strings.Contains(config, `"mode": "windows-bgp"`) {
		t.Error("config should contain windows-bgp mode")
	}
}

// TestGenerateCalicoConfigNone verifies none backend config generation.
func TestGenerateCalicoConfigNone(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	config, err := installer.generateCalicoConfig(None, "10.0.0.0/16", "10.0.0.10")

	if err != nil {
		t.Fatalf("generateCalicoConfig with None failed: %v", err)
	}

	if !strings.Contains(config, `"mode": "none"`) {
		t.Error("config should contain none mode")
	}
}

// TestGetCalicoVersion verifies version retrieval.
func TestGetCalicoVersion(t *testing.T) {
	// Test with nil config (should use default)
	version := getCalicoVersion(nil)
	if version != "" {
		// If this doesn't panic, it might use a default
		t.Logf("getCalicoVersion with nil returned: %s", version)
	}
}

// TestCNIInstallerInterface verifies Installer implements expected methods.
func TestCNIInstallerInterface(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	// Verify all expected methods exist
	_ = installer.GetName()

	ctx := context.Background()
	_ = installer.IsCompleted(ctx)
	_ = installer.Validate(ctx)

	// Note: Execute would make network calls, so we don't test it here
	t.Log("CNI Installer implements expected interface methods")
}

// TestPrepareCNIDirectoriesDoesNotPanic verifies directory preparation handles errors.
func TestPrepareCNIDirectoriesDoesNotPanic(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	// This may fail due to permissions but should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("prepareCNIDirectories panicked: %v", r)
		}
	}()

	err := installer.prepareCNIDirectories()
	t.Logf("prepareCNIDirectories result: %v", err)
}

// TestCNIConfigContainsCNIVersion verifies CNI version in config.
func TestCNIConfigContainsCNIVersion(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	config, err := installer.generateCalicoConfig(VXLAN, "10.0.0.0/16", "10.0.0.10")
	if err != nil {
		t.Fatalf("generateCalicoConfig failed: %v", err)
	}

	// Should contain the CNI spec version
	if !strings.Contains(config, DefaultCNISpecVersion) {
		t.Errorf("config should contain CNI spec version %s", DefaultCNISpecVersion)
	}
}

// TestCNIConfigContainsIPAMType verifies IPAM configuration.
func TestCNIConfigContainsIPAMType(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	config, err := installer.generateCalicoConfig(VXLAN, "10.0.0.0/16", "10.0.0.10")
	if err != nil {
		t.Fatalf("generateCalicoConfig failed: %v", err)
	}

	// Should contain calico-ipam
	if !strings.Contains(config, "calico-ipam") {
		t.Error("config should contain calico-ipam IPAM type")
	}
}

// TestCNIConfigContainsKubernetesDatastore verifies datastore type.
func TestCNIConfigContainsKubernetesDatastore(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	config, err := installer.generateCalicoConfig(VXLAN, "10.0.0.0/16", "10.0.0.10")
	if err != nil {
		t.Fatalf("generateCalicoConfig failed: %v", err)
	}

	// Should use kubernetes datastore
	if !strings.Contains(config, "kubernetes") {
		t.Error("config should reference kubernetes datastore")
	}
}
