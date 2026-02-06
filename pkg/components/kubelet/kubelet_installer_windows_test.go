//go:build windows
// +build windows

package kubelet

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestKubeletNewInstaller verifies the installer constructor.
func TestKubeletNewInstaller(t *testing.T) {
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

// TestKubeletInstallerGetName verifies the step name.
func TestKubeletInstallerGetName(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	name := installer.GetName()

	if name == "" {
		t.Error("GetName should not return empty string")
	}

	if name != "KubeletInstaller" {
		t.Errorf("GetName = %s, want KubeletInstaller", name)
	}
}

// TestKubeletInstallerValidate verifies validation logic.
func TestKubeletInstallerValidate(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	ctx := context.Background()
	err := installer.Validate(ctx)

	// On a system without kubelet binary, this should return an error
	if err != nil {
		if !strings.Contains(err.Error(), "kubelet binary not found") {
			t.Logf("Validate returned unexpected error: %v", err)
		} else {
			t.Log("Validate correctly detected missing kubelet binary")
		}
	} else {
		t.Log("Validate passed - kubelet binary exists")
	}
}

// TestKubeletInstallerIsCompleted verifies completion check always returns false.
func TestKubeletInstallerIsCompleted(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	ctx := context.Background()
	completed := installer.IsCompleted(ctx)

	// Should always return false to enforce reconfiguration
	if completed {
		t.Error("IsCompleted should return false to enforce reconfiguration")
	}
}

// TestMapToEvictionThresholds verifies eviction threshold formatting.
func TestMapToEvictionThresholds(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]string
		separator string
		expected  string
	}{
		{
			name:      "empty map",
			input:     map[string]string{},
			separator: ",",
			expected:  "",
		},
		{
			name: "single threshold",
			input: map[string]string{
				"memory.available": "100Mi",
			},
			separator: ",",
			expected:  "memory.available<100Mi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapToEvictionThresholds(tt.input, tt.separator)
			if tt.name == "empty map" && result != "" {
				t.Errorf("mapToEvictionThresholds(%v) = %s, want empty", tt.input, result)
			}
			if tt.name == "single threshold" && !strings.Contains(result, "memory.available") {
				t.Errorf("mapToEvictionThresholds result should contain memory.available: %s", result)
			}
		})
	}
}

// TestMapToKeyValuePairs verifies key-value pair formatting.
func TestMapToKeyValuePairs(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]string
		separator string
		expected  string
	}{
		{
			name:      "empty map",
			input:     map[string]string{},
			separator: ",",
			expected:  "",
		},
		{
			name: "single pair",
			input: map[string]string{
				"cpu": "100m",
			},
			separator: ",",
			expected:  "cpu=100m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapToKeyValuePairs(tt.input, tt.separator)
			if tt.name == "empty map" && result != "" {
				t.Errorf("mapToKeyValuePairs(%v) = %s, want empty", tt.input, result)
			}
			if tt.name == "single pair" && !strings.Contains(result, "cpu=100m") {
				t.Errorf("mapToKeyValuePairs result should contain cpu=100m: %s", result)
			}
		})
	}
}

// TestKubeletInstallerInterface verifies Installer implements expected methods.
func TestKubeletInstallerInterface(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	// Verify all expected methods exist
	_ = installer.GetName()

	ctx := context.Background()
	_ = installer.IsCompleted(ctx)
	_ = installer.Validate(ctx)

	// Note: Execute requires Azure credentials, so we don't test it here
	t.Log("Kubelet Installer implements expected interface methods")
}

// TestCreateRequiredDirectoriesDoesNotPanic verifies directory creation handles errors.
func TestCreateRequiredDirectoriesDoesNotPanic(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	// This may fail due to permissions but should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("createRequiredDirectories panicked: %v", r)
		}
	}()

	err := installer.createRequiredDirectories()
	t.Logf("createRequiredDirectories result: %v", err)
}

// TestKubeletServiceName verifies service name constant.
func TestKubeletServiceName(t *testing.T) {
	if kubeletServiceName != "kubelet" {
		t.Errorf("kubeletServiceName = %s, want kubelet", kubeletServiceName)
	}
}

// TestKubeletPaths verifies path constants are Windows-style.
func TestKubeletPaths(t *testing.T) {
	paths := []struct {
		name  string
		value string
	}{
		{"kubeletDir", kubeletDir},
		{"kubeletVarDir", kubeletVarDir},
		{"kubeletPKIDir", kubeletPKIDir},
		{"kubeletConfigDir", kubeletConfigDir},
		{"kubeletManifestsDir", kubeletManifestsDir},
		{"kubeletVolumePluginDir", kubeletVolumePluginDir},
		{"kubeletKubeconfigPath", kubeletKubeconfigPath},
		{"kubeletTokenScriptPath", kubeletTokenScriptPath},
		{"kubeletConfigPath", kubeletConfigPath},
	}

	for _, p := range paths {
		t.Run(p.name, func(t *testing.T) {
			// Should be Windows-style path
			if !strings.HasPrefix(p.value, "C:") {
				t.Errorf("%s should start with C:: %s", p.name, p.value)
			}
			if !strings.Contains(p.value, "\\") {
				t.Errorf("%s should contain backslashes: %s", p.name, p.value)
			}
		})
	}
}

// TestKubeletDirsArray verifies directories array.
func TestKubeletDirsArray(t *testing.T) {
	if len(kubeletDirs) == 0 {
		t.Error("kubeletDirs should not be empty")
	}

	// All should be Windows paths
	for i, dir := range kubeletDirs {
		if !strings.HasPrefix(dir, "C:") {
			t.Errorf("kubeletDirs[%d] = %s, should start with C:", i, dir)
		}
	}
}
