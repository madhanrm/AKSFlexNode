//go:build windows
// +build windows

package cni

const (
	// CNI directories for Windows (aligned with AKS Arc patterns)
	DefaultCNIBinDir  = "C:\\k\\cni"
	DefaultCNIConfDir = "C:\\k\\cni\\config"
	DefaultCNILibDir  = "C:\\var\\lib\\cni"

	// Calico directories (aligned with AKS Arc)
	CalicoDir     = "C:\\CalicoWindows"
	CalicoDataDir = "C:\\var\\lib\\calico"
	CalicoLogDir  = "C:\\var\\log\\calico"
	CalicoEtcDir  = "C:\\etc\\CalicoWindows"

	// CNI configuration files
	calicoConfigFile = "10-calico.conf"

	// Required CNI plugins for Calico on Windows
	calicoPlugin     = "calico.exe"
	calicoIPAMPlugin = "calico-ipam.exe"
	hostLocalPlugin  = "host-local.exe"
	winBridgePlugin  = "win-bridge.exe"
	winOverlayPlugin = "win-overlay.exe"
	flannelPlugin    = "flannel.exe"

	// Calico version - aligned with AKS Arc (v6 release stream)
	DefaultCalicoVersion = "3.28.2"

	// CNI specification version for configuration files
	DefaultCNISpecVersion = "0.3.1"

	// Calico HostProcess container image (used by AKS Arc)
	// The actual CNI setup is done by the Calico DaemonSet running as HostProcess container
	CalicoHostProcessImage = "mcr.microsoft.com/aksarc/calico-windows"
)

var cniDirs = []string{
	DefaultCNIBinDir,
	DefaultCNIConfDir,
	DefaultCNILibDir,
	CalicoDir,
	CalicoDataDir,
	CalicoLogDir,
	CalicoEtcDir,
}

// Required Calico CNI plugins
var requiredCNIPlugins = []string{
	calicoPlugin,
	calicoIPAMPlugin,
}

// Calico for Windows download URLs
// Primary: Azure Kubernetes artifacts CDN (used by AKS)
// Fallback: Official Tigera/Calico releases
var (
	// AKS Calico Windows package URL (from k8sreleases blob storage)
	calicoWindowsZipURL = "https://k8sreleases.blob.core.windows.net/calico-node/v%s/binaries/calico-windows-v%s.zip"

	// Fallback: Official Calico releases from GitHub
	calicoGitHubZipURL = "https://github.com/projectcalico/calico/releases/download/v%s/calico-windows-v%s.zip"
)

// NetworkingBackend represents the Calico networking backend
type NetworkingBackend string

const (
	// VXLAN uses VXLAN overlay networking (default for AKS Arc)
	VXLAN NetworkingBackend = "vxlan"
	// WindowsBGP uses BGP for routing (requires BGP-capable infrastructure)
	WindowsBGP NetworkingBackend = "windows-bgp"
	// None disables Calico CNI plugin (to use another CNI)
	None NetworkingBackend = "none"
)
