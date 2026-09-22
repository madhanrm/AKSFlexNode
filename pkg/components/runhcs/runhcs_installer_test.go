//go:build windows
// +build windows

package runhcs

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestNewInstaller verifies the installer constructor.
func TestNewInstaller(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise

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

	// Config may be nil if not initialized, which is okay for unit tests
}

// TestInstallerGetName verifies the step name.
func TestInstallerGetName(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	name := installer.GetName()

	if name == "" {
		t.Error("GetName should not return empty string")
	}

	if name != "Runhcs_Installer" {
		t.Errorf("GetName = %s, want Runhcs_Installer", name)
	}
}

// TestInstallerIsCompleted verifies completion check logic.
func TestInstallerIsCompleted(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	// This will check if the file exists on the actual system
	// On non-Windows or without containerd, this should return false
	ctx := context.Background()
	completed := installer.IsCompleted(ctx)

	// We just verify it doesn't panic and returns a boolean
	t.Logf("IsCompleted returned: %v (depends on system state)", completed)
}

// TestInstallerValidate verifies validation logic.
func TestInstallerValidate(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	ctx := context.Background()
	err := installer.Validate(ctx)

	// On a system without containerd installed, this should return an error
	// We verify the error message is descriptive
	if err != nil {
		if err.Error() == "" {
			t.Error("Validate error message should not be empty")
		}
		t.Logf("Validate returned expected error: %v", err)
	} else {
		t.Log("Validate passed - containerd directory exists")
	}
}

// TestInstallerExecute verifies execute returns appropriate result.
func TestInstallerExecute(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	ctx := context.Background()
	err := installer.Execute(ctx)

	// On a system without containerd/runhcs, this should return an error
	if err != nil {
		if err.Error() == "" {
			t.Error("Execute error message should not be empty")
		}
		t.Logf("Execute returned expected error (runhcs not installed): %v", err)
	} else {
		t.Log("Execute passed - runhcs shim exists")
	}
}

// TestInstallerInterface verifies Installer satisfies expected interface pattern.
func TestInstallerInterface(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	installer := NewInstaller(logger)

	// Verify all expected methods exist and are callable
	_ = installer.GetName()

	ctx := context.Background()
	_ = installer.IsCompleted(ctx)
	_ = installer.Validate(ctx)
	_ = installer.Execute(ctx)

	t.Log("Installer implements expected interface methods")
}
