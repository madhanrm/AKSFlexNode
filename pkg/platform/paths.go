package platform

// PathConfig contains OS-specific paths used by AKS Flex Node components
type PathConfig struct {
	// Container runtime paths
	ContainerdBinDir    string // Directory containing containerd binaries
	ContainerdConfigDir string // Directory for containerd configuration
	ContainerdDataDir   string // Directory for containerd data/state
	ContainerdSocketDir string // Directory for containerd socket

	// Kubernetes paths
	KubeletBinDir     string // Directory containing kubelet binary
	KubeletConfigDir  string // Directory for kubelet configuration
	KubeletDataDir    string // Directory for kubelet data/state
	KubeletManifests  string // Directory for static pod manifests
	KubeletVolumeDir  string // Directory for volume plugins
	KubeletServiceDir string // Directory for kubelet service drop-ins

	// CNI paths
	CNIBinDir  string // Directory containing CNI plugin binaries
	CNIConfDir string // Directory for CNI configuration files

	// System paths
	SystemBinDir    string // System binary directory (/usr/bin or C:\Windows\System32)
	SystemConfigDir string // System configuration directory (/etc or C:\ProgramData)
	SystemDataDir   string // System data directory (/var/lib or C:\ProgramData)
	SystemLogDir    string // System log directory (/var/log or C:\ProgramData\logs)
	TempDir         string // Temporary directory

	// Service paths
	ServiceDir     string // Service definition directory (systemd or Windows Services)
	ServiceConfDir string // Service configuration directory (/etc/default or registry)

	// Azure Arc paths
	ArcAgentBinDir  string // Directory containing Arc agent binary
	ArcAgentDataDir string // Directory for Arc agent data

	// File extensions (platform-specific)
	ExecutableExt string // Executable extension ("" on Linux, ".exe" on Windows)
	ArchiveExt    string // Archive extension (".tar.gz" on Linux, ".zip" on Windows)
	ServiceExt    string // Service file extension (".service" on Linux, "" on Windows)
}

// ContainerdBinaryPath returns the full path to the containerd binary
func (p *PathConfig) ContainerdBinaryPath() string {
	return p.ContainerdBinDir + "/containerd" + p.ExecutableExt
}

// KubeletBinaryPath returns the full path to the kubelet binary
func (p *PathConfig) KubeletBinaryPath() string {
	return p.KubeletBinDir + "/kubelet" + p.ExecutableExt
}

// KubectlBinaryPath returns the full path to the kubectl binary
func (p *PathConfig) KubectlBinaryPath() string {
	return p.KubeletBinDir + "/kubectl" + p.ExecutableExt
}

// KubeadmBinaryPath returns the full path to the kubeadm binary
func (p *PathConfig) KubeadmBinaryPath() string {
	return p.KubeletBinDir + "/kubeadm" + p.ExecutableExt
}

// RuncBinaryPath returns the full path to the runc binary
func (p *PathConfig) RuncBinaryPath() string {
	return p.SystemBinDir + "/runc" + p.ExecutableExt
}

// ContainerdConfigPath returns the full path to the containerd config file
func (p *PathConfig) ContainerdConfigPath() string {
	return p.ContainerdConfigDir + "/config.toml"
}

// ContainerdServicePath returns the full path to the containerd service file
func (p *PathConfig) ContainerdServicePath() string {
	if p.ServiceExt != "" {
		return p.ServiceDir + "/containerd" + p.ServiceExt
	}
	return p.ServiceDir + "/containerd.service"
}

// KubeletServicePath returns the full path to the kubelet service file
func (p *PathConfig) KubeletServicePath() string {
	if p.ServiceExt != "" {
		return p.ServiceDir + "/kubelet" + p.ServiceExt
	}
	return p.ServiceDir + "/kubelet.service"
}

// KubeletKubeconfigPath returns the full path to the kubelet kubeconfig
func (p *PathConfig) KubeletKubeconfigPath() string {
	return p.KubeletDataDir + "/kubeconfig"
}

// KubeletTokenScriptPath returns the full path to the Arc token script
func (p *PathConfig) KubeletTokenScriptPath() string {
	if p.ExecutableExt == ".exe" {
		return p.KubeletDataDir + "/token.ps1"
	}
	return p.KubeletDataDir + "/token.sh"
}

// KubeletDefaultsPath returns the full path to the kubelet defaults file
func (p *PathConfig) KubeletDefaultsPath() string {
	return p.ServiceConfDir + "/kubelet"
}

// Join creates a path by joining components with the appropriate separator
func (p *PathConfig) Join(elem ...string) string {
	if len(elem) == 0 {
		return ""
	}

	// Use forward slashes for Linux, backslashes for Windows
	sep := "/"
	if p.ExecutableExt == ".exe" {
		sep = "\\"
	}

	result := elem[0]
	for _, e := range elem[1:] {
		result = result + sep + e
	}
	return result
}
