//go:build windows
// +build windows

package cni

import (
	"strings"
	"testing"
)

// TestWindowsCNIConstants verifies Windows CNI configuration constants.
func TestWindowsCNIConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"DefaultCNIBinDir", DefaultCNIBinDir, "C:\\k\\cni"},
		{"DefaultCNIConfDir", DefaultCNIConfDir, "C:\\k\\cni\\config"},
		{"DefaultCNILibDir", DefaultCNILibDir, "C:\\var\\lib\\cni"},
		{"CalicoDir", CalicoDir, "C:\\CalicoWindows"},
		{"CalicoDataDir", CalicoDataDir, "C:\\var\\lib\\calico"},
		{"CalicoLogDir", CalicoLogDir, "C:\\var\\log\\calico"},
		{"CalicoEtcDir", CalicoEtcDir, "C:\\etc\\CalicoWindows"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %s, want %s", tt.name, tt.value, tt.expected)
			}
		})
	}
}

// TestCalicoVersionConstants verifies Calico version constants.
func TestCalicoVersionConstants(t *testing.T) {
	if DefaultCalicoVersion == "" {
		t.Error("DefaultCalicoVersion should not be empty")
	}

	// Version should be in semver format
	if !strings.Contains(DefaultCalicoVersion, ".") {
		t.Errorf("DefaultCalicoVersion should be in semver format: %s", DefaultCalicoVersion)
	}

	// Verify we're using a recent version
	if !strings.HasPrefix(DefaultCalicoVersion, "3.") {
		t.Errorf("DefaultCalicoVersion should be 3.x: %s", DefaultCalicoVersion)
	}

	// Current expected version
	if DefaultCalicoVersion != "3.28.2" {
		t.Errorf("DefaultCalicoVersion = %s, want 3.28.2", DefaultCalicoVersion)
	}
}

// TestCalicoURLConstants verifies Calico download URL patterns.
func TestCalicoURLConstants(t *testing.T) {
	if calicoWindowsZipURL == "" {
		t.Error("calicoWindowsZipURL should not be empty")
	}

	// Should be HTTPS
	if !strings.HasPrefix(calicoWindowsZipURL, "https://") {
		t.Errorf("calicoWindowsZipURL should use HTTPS: %s", calicoWindowsZipURL)
	}

	// Should contain calico reference
	if !strings.Contains(strings.ToLower(calicoWindowsZipURL), "calico") {
		t.Errorf("calicoWindowsZipURL should reference calico: %s", calicoWindowsZipURL)
	}

	// Should have version placeholder
	if !strings.Contains(calicoWindowsZipURL, "%s") {
		t.Errorf("calicoWindowsZipURL should have version placeholder: %s", calicoWindowsZipURL)
	}

	// Test GitHub fallback URL
	if calicoGitHubZipURL == "" {
		t.Error("calicoGitHubZipURL should not be empty")
	}
	if !strings.Contains(calicoGitHubZipURL, "github.com") {
		t.Errorf("calicoGitHubZipURL should reference github.com: %s", calicoGitHubZipURL)
	}
}

// TestWindowsCNIDirectories verifies Windows CNI directories array.
func TestWindowsCNIDirectories(t *testing.T) {
	expectedCount := 7 // bin, conf, lib, calico, data, log, etc
	if len(cniDirs) != expectedCount {
		t.Errorf("Expected %d CNI directories, got %d", expectedCount, len(cniDirs))
	}

	// All directories should be Windows-style paths
	for i, dir := range cniDirs {
		if !strings.Contains(dir, "\\") {
			t.Errorf("cniDirs[%d] = %s, should be Windows-style path with backslash", i, dir)
		}
		if !strings.HasPrefix(dir, "C:") {
			t.Errorf("cniDirs[%d] = %s, should start with C:", i, dir)
		}
	}
}

// TestCalicoConfigFileConstants verifies Calico configuration file constants.
func TestCalicoConfigFileConstants(t *testing.T) {
	// Check CNI config file name
	if calicoConfigFile == "" {
		t.Error("calicoConfigFile should not be empty")
	}

	// Should be a .conf file
	if !strings.HasSuffix(calicoConfigFile, ".conf") {
		t.Errorf("calicoConfigFile should be .conf file: %s", calicoConfigFile)
	}

	// Should have priority number prefix (10-)
	if !strings.HasPrefix(calicoConfigFile, "10-") {
		t.Errorf("calicoConfigFile should have 10- prefix: %s", calicoConfigFile)
	}
}

// TestCalicoPluginConstants verifies Calico plugin name constants.
func TestCalicoPluginConstants(t *testing.T) {
	if calicoPlugin == "" {
		t.Error("calicoPlugin should not be empty")
	}

	// Windows plugins should have .exe extension
	if !strings.HasSuffix(calicoPlugin, ".exe") {
		t.Errorf("calicoPlugin should have .exe extension: %s", calicoPlugin)
	}

	// Check IPAM plugin
	if calicoIPAMPlugin == "" {
		t.Error("calicoIPAMPlugin should not be empty")
	}
	if !strings.HasSuffix(calicoIPAMPlugin, ".exe") {
		t.Errorf("calicoIPAMPlugin should have .exe extension: %s", calicoIPAMPlugin)
	}
}

// TestRequiredCNIPlugins verifies required plugins list.
func TestRequiredCNIPlugins(t *testing.T) {
	if len(requiredCNIPlugins) < 2 {
		t.Errorf("Expected at least 2 required CNI plugins, got %d", len(requiredCNIPlugins))
	}

	// Should contain calico and calico-ipam
	hasCalico := false
	hasIPAM := false
	for _, plugin := range requiredCNIPlugins {
		if strings.Contains(plugin, "calico") && !strings.Contains(plugin, "ipam") {
			hasCalico = true
		}
		if strings.Contains(plugin, "ipam") {
			hasIPAM = true
		}
	}

	if !hasCalico {
		t.Error("requiredCNIPlugins should contain calico plugin")
	}
	if !hasIPAM {
		t.Error("requiredCNIPlugins should contain calico-ipam plugin")
	}
}

// TestNetworkingBackendConstants verifies networking backend constants.
func TestNetworkingBackendConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    NetworkingBackend
		expected string
	}{
		{"VXLAN", VXLAN, "vxlan"},
		{"WindowsBGP", WindowsBGP, "windows-bgp"},
		{"None", None, "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("%s = %s, want %s", tt.name, tt.value, tt.expected)
			}
		})
	}
}

// TestCalicoHostProcessImage verifies the HostProcess container image.
func TestCalicoHostProcessImage(t *testing.T) {
	if CalicoHostProcessImage == "" {
		t.Error("CalicoHostProcessImage should not be empty")
	}

	// Should be from MCR
	if !strings.HasPrefix(CalicoHostProcessImage, "mcr.microsoft.com/") {
		t.Errorf("CalicoHostProcessImage should be from MCR: %s", CalicoHostProcessImage)
	}

	// Should reference AKS Arc
	if !strings.Contains(CalicoHostProcessImage, "aksarc") {
		t.Errorf("CalicoHostProcessImage should reference aksarc: %s", CalicoHostProcessImage)
	}
}

// TestDefaultCNISpecVersion verifies CNI spec version.
func TestDefaultCNISpecVersion(t *testing.T) {
	if DefaultCNISpecVersion == "" {
		t.Error("DefaultCNISpecVersion should not be empty")
	}

	// Should be semver format
	if !strings.Contains(DefaultCNISpecVersion, ".") {
		t.Errorf("DefaultCNISpecVersion should be semver: %s", DefaultCNISpecVersion)
	}

	// Version 0.3.1 is expected for compatibility
	if DefaultCNISpecVersion != "0.3.1" {
		t.Errorf("DefaultCNISpecVersion = %s, want 0.3.1", DefaultCNISpecVersion)
	}
}
