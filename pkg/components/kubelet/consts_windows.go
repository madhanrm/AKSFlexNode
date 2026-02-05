//go:build windows
// +build windows

package kubelet

const (
	// Windows kubelet directories (aligned with AKS Arc)
	kubeletDir            = "C:\\k"
	kubeletVarDir         = "C:\\var\\lib\\kubelet"
	kubeletPKIDir         = "C:\\var\\lib\\kubelet\\pki"
	kubeletConfigDir      = "C:\\etc\\kubernetes"
	kubeletManifestsDir   = "C:\\etc\\kubernetes\\manifests"
	kubeletVolumePluginDir = "C:\\etc\\kubernetes\\volumeplugins"

	// Configuration file paths
	kubeletKubeconfigPath  = "C:\\var\\lib\\kubelet\\kubeconfig"
	kubeletTokenScriptPath = "C:\\var\\lib\\kubelet\\token.ps1"
	kubeletConfigPath      = "C:\\var\\lib\\kubelet\\config.yaml"

	// Service configuration
	kubeletServiceName = "kubelet"

	// Azure resource identifiers
	aksServiceResourceID = "6dae42f8-4368-4678-94ff-3960e28e3630"
)

// Windows kubelet directories to create
var kubeletDirs = []string{
	kubeletDir,
	kubeletVarDir,
	kubeletPKIDir,
	kubeletConfigDir,
	kubeletManifestsDir,
	kubeletVolumePluginDir,
}
