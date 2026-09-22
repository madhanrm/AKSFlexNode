package containerd

import (
	"runtime"

	"go.goms.io/aks/AKSFlexNode/pkg/platform"
)

// Platform-specific paths - resolved at runtime
var (
	systemBinDir               string
	defaultContainerdBinaryDir string
	defaultContainerdConfigDir string
	containerdConfigFile       string
	containerdServiceFile      string
	containerdDataDir          string
	containerdDirs             []string
	containerdBinaries         []string
	containerdFileName         string
	containerdDownloadURL      string
)

func init() {
	paths := platform.Current().Paths()

	if runtime.GOOS == "windows" {
		// Windows paths
		systemBinDir = paths.ContainerdBinDir
		defaultContainerdBinaryDir = paths.ContainerdBinDir + "\\containerd.exe"
		defaultContainerdConfigDir = paths.ContainerdConfigDir
		containerdConfigFile = paths.ContainerdConfigDir + "\\config.toml"
		containerdServiceFile = "" // Windows uses SCM, not service files
		containerdDataDir = paths.ContainerdDataDir

		containerdDirs = []string{
			paths.ContainerdConfigDir,
			paths.ContainerdBinDir,
			paths.ContainerdDataDir,
		}

		containerdBinaries = []string{
			"ctr.exe",
			"containerd.exe",
			"containerd-shim-runhcs-v1.exe",
		}

		containerdFileName = "containerd-%s-windows-amd64.tar.gz"
		containerdDownloadURL = "https://github.com/containerd/containerd/releases/download/v%s/" + containerdFileName
	} else {
		// Linux paths (default)
		systemBinDir = "/usr/bin"
		defaultContainerdBinaryDir = "/usr/bin/containerd"
		defaultContainerdConfigDir = "/etc/containerd"
		containerdConfigFile = "/etc/containerd/config.toml"
		containerdServiceFile = "/etc/systemd/system/containerd.service"
		containerdDataDir = "/var/lib/containerd"

		containerdDirs = []string{
			defaultContainerdConfigDir,
		}

		containerdBinaries = []string{
			"ctr",
			"containerd",
			"containerd-shim",
			"containerd-shim-runc-v1",
			"containerd-shim-runc-v2",
			"containerd-stress",
		}

		containerdFileName = "containerd-%s-linux-%s.tar.gz"
		containerdDownloadURL = "https://github.com/containerd/containerd/releases/download/v%s/" + containerdFileName
	}
}
