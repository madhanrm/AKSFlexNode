//go:build windows
// +build windows

package kubelet

import (
	"strings"
	"testing"
)

// TestWindowsKubeletDirectoryConstants verifies Windows kubelet directory constants.
func TestWindowsKubeletDirectoryConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"kubeletDir", kubeletDir, "C:\\k"},
		{"kubeletVarDir", kubeletVarDir, "C:\\var\\lib\\kubelet"},
		{"kubeletPKIDir", kubeletPKIDir, "C:\\var\\lib\\kubelet\\pki"},
		{"kubeletConfigDir", kubeletConfigDir, "C:\\etc\\kubernetes"},
		{"kubeletManifestsDir", kubeletManifestsDir, "C:\\etc\\kubernetes\\manifests"},
		{"kubeletVolumePluginDir", kubeletVolumePluginDir, "C:\\etc\\kubernetes\\volumeplugins"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %s, want %s", tt.name, tt.value, tt.expected)
			}
		})
	}
}

// TestWindowsKubeletConfigPaths verifies kubelet configuration file paths.
func TestWindowsKubeletConfigPaths(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"kubeletKubeconfigPath", kubeletKubeconfigPath, "C:\\var\\lib\\kubelet\\kubeconfig"},
		{"kubeletTokenScriptPath", kubeletTokenScriptPath, "C:\\var\\lib\\kubelet\\token.ps1"},
		{"kubeletConfigPath", kubeletConfigPath, "C:\\var\\lib\\kubelet\\config.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %s, want %s", tt.name, tt.value, tt.expected)
			}
		})
	}
}

// TestWindowsKubeletServiceName verifies kubelet service name.
func TestWindowsKubeletServiceName(t *testing.T) {
	if kubeletServiceName != "kubelet" {
		t.Errorf("kubeletServiceName = %s, want kubelet", kubeletServiceName)
	}
}

// TestAKSServiceResourceID verifies AKS service resource ID is valid GUID format.
func TestAKSServiceResourceID(t *testing.T) {
	if aksServiceResourceID == "" {
		t.Error("aksServiceResourceID should not be empty")
	}

	// Should be a valid GUID format (8-4-4-4-12 hex digits)
	parts := strings.Split(aksServiceResourceID, "-")
	if len(parts) != 5 {
		t.Errorf("aksServiceResourceID should be GUID format (5 parts): %s", aksServiceResourceID)
	}

	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			t.Errorf("aksServiceResourceID part %d should have %d chars, got %d: %s",
				i, expectedLengths[i], len(part), part)
		}
	}
}

// TestWindowsKubeletDirectoriesArray verifies kubelet directories array.
func TestWindowsKubeletDirectoriesArray(t *testing.T) {
	expectedCount := 6
	if len(kubeletDirs) != expectedCount {
		t.Errorf("Expected %d kubelet directories, got %d", expectedCount, len(kubeletDirs))
	}

	// All directories should be Windows-style paths
	for i, dir := range kubeletDirs {
		if !strings.Contains(dir, "\\") {
			t.Errorf("kubeletDirs[%d] = %s, should be Windows-style path with backslash", i, dir)
		}
		if !strings.HasPrefix(dir, "C:") {
			t.Errorf("kubeletDirs[%d] = %s, should start with C:", i, dir)
		}
	}
}

// TestKubeletDirsContainsRequiredDirectories verifies all required directories are in the list.
func TestKubeletDirsContainsRequiredDirectories(t *testing.T) {
	required := map[string]bool{
		kubeletDir:             false,
		kubeletVarDir:          false,
		kubeletPKIDir:          false,
		kubeletConfigDir:       false,
		kubeletManifestsDir:    false,
		kubeletVolumePluginDir: false,
	}

	for _, dir := range kubeletDirs {
		if _, exists := required[dir]; exists {
			required[dir] = true
		}
	}

	for dir, found := range required {
		if !found {
			t.Errorf("kubeletDirs should contain %s", dir)
		}
	}
}

// TestKubeletPathsAreUnderKubeletDir verifies path hierarchy.
func TestKubeletPathsAreUnderKubeletDir(t *testing.T) {
	// kubeletVarDir should be under C:\var
	if !strings.HasPrefix(kubeletVarDir, "C:\\var") {
		t.Errorf("kubeletVarDir should be under C:\\var: %s", kubeletVarDir)
	}

	// PKI dir should be under var dir
	if !strings.HasPrefix(kubeletPKIDir, kubeletVarDir) {
		t.Errorf("kubeletPKIDir should be under kubeletVarDir: %s", kubeletPKIDir)
	}

	// Config paths should be under config dir or var dir
	if !strings.HasPrefix(kubeletKubeconfigPath, kubeletVarDir) {
		t.Errorf("kubeletKubeconfigPath should be under kubeletVarDir: %s", kubeletKubeconfigPath)
	}

	if !strings.HasPrefix(kubeletTokenScriptPath, kubeletVarDir) {
		t.Errorf("kubeletTokenScriptPath should be under kubeletVarDir: %s", kubeletTokenScriptPath)
	}

	// Manifests should be under config dir
	if !strings.HasPrefix(kubeletManifestsDir, kubeletConfigDir) {
		t.Errorf("kubeletManifestsDir should be under kubeletConfigDir: %s", kubeletManifestsDir)
	}
}

// TestTokenScriptPathHasPowerShellExtension verifies token script is PowerShell.
func TestTokenScriptPathHasPowerShellExtension(t *testing.T) {
	if !strings.HasSuffix(kubeletTokenScriptPath, ".ps1") {
		t.Errorf("kubeletTokenScriptPath should have .ps1 extension: %s", kubeletTokenScriptPath)
	}
}

// TestKubeletConfigPathHasYamlExtension verifies config is YAML format.
func TestKubeletConfigPathHasYamlExtension(t *testing.T) {
	if !strings.HasSuffix(kubeletConfigPath, ".yaml") && !strings.HasSuffix(kubeletConfigPath, ".yml") {
		t.Errorf("kubeletConfigPath should have .yaml or .yml extension: %s", kubeletConfigPath)
	}
}
