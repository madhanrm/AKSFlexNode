package platform

import (
	"runtime"
	"strings"
	"testing"
)

// TestPathConfigNotNil verifies PathConfig is always initialized.
func TestPathConfigNotNil(t *testing.T) {
	p := Current()
	paths := p.Paths()
	if paths == nil {
		t.Fatal("Paths() should not return nil")
	}
}

// TestPathConfigFields verifies all PathConfig fields are populated.
func TestPathConfigFields(t *testing.T) {
	p := Current()
	paths := p.Paths()

	// Verify all fields are non-empty
	fields := map[string]string{
		"ContainerdBinDir":    paths.ContainerdBinDir,
		"ContainerdConfigDir": paths.ContainerdConfigDir,
		"ContainerdDataDir":   paths.ContainerdDataDir,
		"KubeletBinDir":       paths.KubeletBinDir,
		"KubeletConfigDir":    paths.KubeletConfigDir,
		"KubeletDataDir":      paths.KubeletDataDir,
		"CNIBinDir":           paths.CNIBinDir,
		"CNIConfDir":          paths.CNIConfDir,
		"SystemBinDir":        paths.SystemBinDir,
		"SystemConfigDir":     paths.SystemConfigDir,
		"TempDir":             paths.TempDir,
		"ServiceDir":          paths.ServiceDir,
	}

	for name, value := range fields {
		if value == "" {
			t.Errorf("PathConfig.%s should not be empty", name)
		}
	}
}

// TestPathConfigLinux verifies Linux path values.
func TestPathConfigLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux path test on non-Linux system")
	}

	p := Current()
	paths := p.Paths()

	// Linux paths should use forward slashes
	if !strings.HasPrefix(paths.ContainerdBinDir, "/") {
		t.Errorf("Linux ContainerdBinDir should start with /, got %s", paths.ContainerdBinDir)
	}

	// Verify specific Linux paths
	tests := []struct {
		name     string
		value    string
		contains string
	}{
		{"ContainerdBinDir", paths.ContainerdBinDir, "/"},
		{"ContainerdConfigDir", paths.ContainerdConfigDir, "/etc"},
		{"KubeletBinDir", paths.KubeletBinDir, "/"},
		{"CNIBinDir", paths.CNIBinDir, "/cni"},
		{"CNIConfDir", paths.CNIConfDir, "/cni"},
		{"ServiceDir", paths.ServiceDir, "/systemd"},
		{"TempDir", paths.TempDir, "/tmp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.value, tt.contains) {
				t.Errorf("%s = %s, want to contain %s", tt.name, tt.value, tt.contains)
			}
		})
	}
}

// TestPathConfigWindows verifies Windows path values.
func TestPathConfigWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows path test on non-Windows system")
	}

	p := Current()
	paths := p.Paths()

	// Windows paths should typically start with a drive letter
	if !strings.Contains(paths.ContainerdBinDir, ":\\") && !strings.HasPrefix(paths.ContainerdBinDir, "C:\\") {
		t.Logf("Windows ContainerdBinDir: %s (may use different format)", paths.ContainerdBinDir)
	}

	// Verify specific Windows paths patterns
	tests := []struct {
		name     string
		value    string
		contains string
	}{
		{"ContainerdBinDir", paths.ContainerdBinDir, "containerd"},
		{"KubeletBinDir", paths.KubeletBinDir, "k"},
		{"CNIBinDir", paths.CNIBinDir, "cni"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lowerValue := strings.ToLower(tt.value)
			lowerContains := strings.ToLower(tt.contains)
			if !strings.Contains(lowerValue, lowerContains) {
				t.Errorf("%s = %s, want to contain %s (case insensitive)", tt.name, tt.value, tt.contains)
			}
		})
	}
}

// TestPathConfigConsistency verifies related paths are consistent.
func TestPathConfigConsistency(t *testing.T) {
	p := Current()
	paths := p.Paths()

	// CNI paths should be related
	if !strings.Contains(strings.ToLower(paths.CNIBinDir), "cni") {
		t.Errorf("CNIBinDir should contain 'cni': %s", paths.CNIBinDir)
	}
	if !strings.Contains(strings.ToLower(paths.CNIConfDir), "cni") {
		t.Errorf("CNIConfDir should contain 'cni': %s", paths.CNIConfDir)
	}

	// Containerd paths should be related
	if !strings.Contains(strings.ToLower(paths.ContainerdBinDir), "containerd") &&
		!strings.Contains(strings.ToLower(paths.ContainerdBinDir), "bin") {
		t.Logf("ContainerdBinDir may not follow expected pattern: %s", paths.ContainerdBinDir)
	}
}

// TestExecutableExtension verifies executable extension is correct for OS.
func TestExecutableExtension(t *testing.T) {
	p := Current()
	paths := p.Paths()

	if runtime.GOOS == "windows" {
		if paths.ExecutableExt != ".exe" {
			t.Errorf("ExecutableExt on Windows should be .exe, got %s", paths.ExecutableExt)
		}
	} else {
		if paths.ExecutableExt != "" {
			t.Errorf("ExecutableExt on Linux should be empty, got %s", paths.ExecutableExt)
		}
	}
}

// TestArchiveExtension verifies archive extension is correct for OS.
func TestArchiveExtension(t *testing.T) {
	p := Current()
	paths := p.Paths()

	if runtime.GOOS == "windows" {
		if paths.ArchiveExt != ".zip" {
			t.Errorf("ArchiveExt on Windows should be .zip, got %s", paths.ArchiveExt)
		}
	} else {
		if paths.ArchiveExt != ".tar.gz" {
			t.Errorf("ArchiveExt on Linux should be .tar.gz, got %s", paths.ArchiveExt)
		}
	}
}

// TestPathConfigHelperMethods verifies helper methods work correctly.
func TestPathConfigHelperMethods(t *testing.T) {
	p := Current()
	paths := p.Paths()

	// Test ContainerdBinaryPath
	containerdPath := paths.ContainerdBinaryPath()
	if containerdPath == "" {
		t.Error("ContainerdBinaryPath should not return empty")
	}
	if !strings.Contains(containerdPath, "containerd") {
		t.Errorf("ContainerdBinaryPath should contain 'containerd': %s", containerdPath)
	}

	// Test KubeletBinaryPath
	kubeletPath := paths.KubeletBinaryPath()
	if kubeletPath == "" {
		t.Error("KubeletBinaryPath should not return empty")
	}
	if !strings.Contains(kubeletPath, "kubelet") {
		t.Errorf("KubeletBinaryPath should contain 'kubelet': %s", kubeletPath)
	}

	// Test KubectlBinaryPath
	kubectlPath := paths.KubectlBinaryPath()
	if kubectlPath == "" {
		t.Error("KubectlBinaryPath should not return empty")
	}
	if !strings.Contains(kubectlPath, "kubectl") {
		t.Errorf("KubectlBinaryPath should contain 'kubectl': %s", kubectlPath)
	}
}

// TestPathConfigJoin verifies the Join method.
func TestPathConfigJoin(t *testing.T) {
	p := Current()
	paths := p.Paths()

	// Test joining paths
	result := paths.Join("a", "b", "c")
	if result == "" {
		t.Error("Join should not return empty")
	}

	// Should contain all components
	if !strings.Contains(result, "a") || !strings.Contains(result, "b") || !strings.Contains(result, "c") {
		t.Errorf("Join result should contain all components: %s", result)
	}

	// Test empty join
	emptyResult := paths.Join()
	if emptyResult != "" {
		t.Errorf("Join with no args should return empty, got %s", emptyResult)
	}

	// Test single element
	singleResult := paths.Join("single")
	if singleResult != "single" {
		t.Errorf("Join with single arg should return that arg, got %s", singleResult)
	}
}
