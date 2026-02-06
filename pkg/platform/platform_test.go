package platform

import (
	"runtime"
	"testing"
)

// TestOSConstants verifies OS type constants are defined correctly.
func TestOSConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    OS
		expected string
	}{
		{"Linux", Linux, "linux"},
		{"Windows", Windows, "windows"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("%s = %s, want %s", tt.name, tt.value, tt.expected)
			}
		})
	}
}

// TestRestartPolicyConstants verifies restart policy constants.
func TestRestartPolicyConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    RestartPolicy
		expected string
	}{
		{"RestartAlways", RestartAlways, "always"},
		{"RestartOnFailure", RestartOnFailure, "on-failure"},
		{"RestartNever", RestartNever, "never"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("%s = %s, want %s", tt.name, tt.value, tt.expected)
			}
		})
	}
}

// TestIsLinux verifies IsLinux returns correct value based on runtime.
func TestIsLinux(t *testing.T) {
	expected := runtime.GOOS == "linux"
	if IsLinux() != expected {
		t.Errorf("IsLinux() = %v, want %v (GOOS=%s)", IsLinux(), expected, runtime.GOOS)
	}
}

// TestIsWindows verifies IsWindows returns correct value based on runtime.
func TestIsWindows(t *testing.T) {
	expected := runtime.GOOS == "windows"
	if IsWindows() != expected {
		t.Errorf("IsWindows() = %v, want %v (GOOS=%s)", IsWindows(), expected, runtime.GOOS)
	}
}

// TestCurrent verifies Current returns a valid platform instance.
func TestCurrent(t *testing.T) {
	p := Current()
	if p == nil {
		t.Fatal("Current() should not return nil")
	}

	// Verify OS matches runtime
	expectedOS := Linux
	if runtime.GOOS == "windows" {
		expectedOS = Windows
	}
	if p.OS() != expectedOS {
		t.Errorf("Current().OS() = %v, want %v", p.OS(), expectedOS)
	}

	// Verify all interface methods return non-nil
	if p.Paths() == nil {
		t.Error("Current().Paths() should not return nil")
	}
	if p.Service() == nil {
		t.Error("Current().Service() should not return nil")
	}
	if p.Command() == nil {
		t.Error("Current().Command() should not return nil")
	}
	if p.FileSystem() == nil {
		t.Error("Current().FileSystem() should not return nil")
	}
}

// TestCurrentSingleton verifies Current returns the same instance.
func TestCurrentSingleton(t *testing.T) {
	p1 := Current()
	p2 := Current()
	if p1 != p2 {
		t.Error("Current() should return the same singleton instance")
	}
}

// TestServiceConfig verifies ServiceConfig struct fields.
func TestServiceConfig(t *testing.T) {
	config := &ServiceConfig{
		Name:           "test-service",
		DisplayName:    "Test Service",
		Description:    "A test service",
		BinaryPath:     "/usr/bin/test",
		Args:           []string{"--arg1", "--arg2"},
		WorkingDir:     "/var/lib/test",
		Dependencies:   []string{"network.target"},
		Environment:    map[string]string{"KEY": "value"},
		User:           "root",
		RestartPolicy:  RestartAlways,
		RestartDelayMs: 5000,
	}

	if config.Name != "test-service" {
		t.Error("ServiceConfig.Name not set correctly")
	}
	if config.DisplayName != "Test Service" {
		t.Error("ServiceConfig.DisplayName not set correctly")
	}
	if config.Description != "A test service" {
		t.Error("ServiceConfig.Description not set correctly")
	}
	if config.BinaryPath != "/usr/bin/test" {
		t.Error("ServiceConfig.BinaryPath not set correctly")
	}
	if len(config.Args) != 2 {
		t.Error("ServiceConfig.Args not set correctly")
	}
	if config.WorkingDir != "/var/lib/test" {
		t.Error("ServiceConfig.WorkingDir not set correctly")
	}
	if len(config.Dependencies) != 1 {
		t.Error("ServiceConfig.Dependencies not set correctly")
	}
	if config.Environment["KEY"] != "value" {
		t.Error("ServiceConfig.Environment not set correctly")
	}
	if config.User != "root" {
		t.Error("ServiceConfig.User not set correctly")
	}
	if config.RestartPolicy != RestartAlways {
		t.Error("ServiceConfig.RestartPolicy not set correctly")
	}
	if config.RestartDelayMs != 5000 {
		t.Error("ServiceConfig.RestartDelayMs not set correctly")
	}
}
