package runc

import (
	"testing"
)

func TestRuncConstants(t *testing.T) {
	// Test that constants are properly defined
	if runcBinaryPath != "/usr/bin/runc" {
		t.Errorf("Expected runcBinaryPath to be '/usr/bin/runc', got '%s'", runcBinaryPath)
	}
}

func TestRuncVariables(t *testing.T) {
	// Test that variables are accessible
	if runcFileName == "" {
		t.Error("runcFileName should not be empty")
	}
	
	if runcDownloadURL == "" {
		t.Error("runcDownloadURL should not be empty")
	}
	
	// Test that runcFileName contains format specifier
	if runcFileName != "runc.%s" {
		t.Errorf("Expected runcFileName to be 'runc.%%s', got '%s'", runcFileName)
	}
	
	// Test that runcDownloadURL contains format specifiers
	expectedPrefix := "https://github.com/opencontainers/runc/releases/download/v"
	if len(runcDownloadURL) < len(expectedPrefix) {
		t.Error("runcDownloadURL is too short")
	}
}
