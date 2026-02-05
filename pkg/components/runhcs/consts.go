//go:build windows
// +build windows

package runhcs

// Windows container runtime shim paths
const (
	// runhcsBinaryPath is the path to the runhcs binary (included with containerd on Windows)
	runhcsBinaryPath = `C:\Program Files\containerd\bin\containerd-shim-runhcs-v1.exe`
)

var (
	// hcsshimDownloadURL is the download URL for the hcsshim release
	// Note: hcsshim/runhcs is typically bundled with containerd on Windows
	hcsshimFileName    = "containerd-shim-runhcs-v1.exe"
	hcsshimDownloadURL = "https://github.com/microsoft/hcsshim/releases/download/v%s/containerd-shim-runhcs-v1.exe"
)
