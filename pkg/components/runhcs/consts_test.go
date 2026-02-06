//go:build windows
// +build windows

package runhcs

import (
	"strings"
	"testing"
)

// TestRunhcsBinaryPath verifies the runhcs binary path constant.
func TestRunhcsBinaryPath(t *testing.T) {
	expected := `C:\Program Files\containerd\bin\containerd-shim-runhcs-v1.exe`
	if runhcsBinaryPath != expected {
		t.Errorf("runhcsBinaryPath = %s, want %s", runhcsBinaryPath, expected)
	}
}

// TestRunhcsBinaryPathIsWindowsPath verifies the path is Windows-style.
func TestRunhcsBinaryPathIsWindowsPath(t *testing.T) {
	// Should start with drive letter
	if !strings.HasPrefix(runhcsBinaryPath, "C:") {
		t.Errorf("runhcsBinaryPath should start with C:: %s", runhcsBinaryPath)
	}

	// Should use backslashes
	if !strings.Contains(runhcsBinaryPath, "\\") {
		t.Errorf("runhcsBinaryPath should use backslashes: %s", runhcsBinaryPath)
	}

	// Should have .exe extension
	if !strings.HasSuffix(runhcsBinaryPath, ".exe") {
		t.Errorf("runhcsBinaryPath should have .exe extension: %s", runhcsBinaryPath)
	}
}

// TestRunhcsBinaryPathInContainerdDir verifies runhcs is in containerd directory.
func TestRunhcsBinaryPathInContainerdDir(t *testing.T) {
	// Should be in containerd bin directory
	if !strings.Contains(runhcsBinaryPath, "containerd") {
		t.Errorf("runhcsBinaryPath should be in containerd directory: %s", runhcsBinaryPath)
	}

	if !strings.Contains(runhcsBinaryPath, "bin") {
		t.Errorf("runhcsBinaryPath should be in bin directory: %s", runhcsBinaryPath)
	}
}

// TestHcsshimFileName verifies the hcsshim filename.
func TestHcsshimFileName(t *testing.T) {
	if hcsshimFileName == "" {
		t.Error("hcsshimFileName should not be empty")
	}

	// Should be the runhcs shim binary
	if !strings.Contains(hcsshimFileName, "runhcs") {
		t.Errorf("hcsshimFileName should contain runhcs: %s", hcsshimFileName)
	}

	// Should have .exe extension
	if !strings.HasSuffix(hcsshimFileName, ".exe") {
		t.Errorf("hcsshimFileName should have .exe extension: %s", hcsshimFileName)
	}
}

// TestHcsshimDownloadURL verifies the hcsshim download URL pattern.
func TestHcsshimDownloadURL(t *testing.T) {
	if hcsshimDownloadURL == "" {
		t.Error("hcsshimDownloadURL should not be empty")
	}

	// Should use HTTPS
	if !strings.HasPrefix(hcsshimDownloadURL, "https://") {
		t.Errorf("hcsshimDownloadURL should use HTTPS: %s", hcsshimDownloadURL)
	}

	// Should be from GitHub hcsshim repo
	if !strings.Contains(hcsshimDownloadURL, "github.com/microsoft/hcsshim") {
		t.Errorf("hcsshimDownloadURL should be from microsoft/hcsshim: %s", hcsshimDownloadURL)
	}

	// Should have version placeholder
	if !strings.Contains(hcsshimDownloadURL, "%s") {
		t.Errorf("hcsshimDownloadURL should have version placeholder: %s", hcsshimDownloadURL)
	}

	// Should reference the runhcs exe
	if !strings.Contains(hcsshimDownloadURL, "containerd-shim-runhcs") {
		t.Errorf("hcsshimDownloadURL should reference containerd-shim-runhcs: %s", hcsshimDownloadURL)
	}
}

// TestShimBinaryNaming verifies containerd shim binary naming convention.
func TestShimBinaryNaming(t *testing.T) {
	// Containerd shim naming convention: containerd-shim-<runtime>-v<version>
	expectedPrefix := "containerd-shim-runhcs-v"
	if !strings.Contains(runhcsBinaryPath, expectedPrefix) {
		t.Errorf("runhcs binary should follow containerd shim naming convention (%s): %s",
			expectedPrefix, runhcsBinaryPath)
	}

	if !strings.Contains(hcsshimFileName, expectedPrefix) {
		t.Errorf("hcsshim filename should follow containerd shim naming convention (%s): %s",
			expectedPrefix, hcsshimFileName)
	}
}
