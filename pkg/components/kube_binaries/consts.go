package kube_binaries

import (
	"path/filepath"
	"runtime"
)

// Binary names
const (
	kubeletBinary = "kubelet"
	kubectlBinary = "kubectl"
	kubeadmBinary = "kubeadm"
)

// Exported constants for repository management (Linux-specific)
const (
	KubernetesRepoList = "/etc/apt/sources.list.d/kubernetes.list"
	KubernetesKeyring  = "/etc/apt/keyrings/kubernetes-apt-keyring.gpg"
)

var (
	// Binary installation directory
	binDir string

	// Kubernetes binary paths
	kubeletPath string
	kubectlPath string
	kubeadmPath string

	// Download URL templates
	kubernetesFileName           string
	defaultKubernetesURLTemplate string
	kubernetesTarPath            string

	// List of all kube binary paths
	kubeBinariesPaths []string

	// Executable extension
	execExt string
)

func init() {
	if runtime.GOOS == "windows" {
		binDir = `C:\k`
		execExt = ".exe"
		kubernetesFileName = "kubernetes-node-windows-amd64.tar.gz"
		defaultKubernetesURLTemplate = "https://kubernetesartifacts.azureedge.net/kubernetes/v%s/binaries/kubernetes-node-windows-%s.tar.gz"
		kubernetesTarPath = "kubernetes/node/bin/"
	} else {
		binDir = "/usr/local/bin"
		execExt = ""
		kubernetesFileName = "kubernetes-node-linux-%s.tar.gz"
		defaultKubernetesURLTemplate = "https://acs-mirror.azureedge.net/kubernetes/v%s/binaries/kubernetes-node-linux-%s.tar.gz"
		kubernetesTarPath = "kubernetes/node/bin/"
	}

	// Use filepath.Join for proper OS-specific path separators
	kubeletPath = filepath.Join(binDir, kubeletBinary+execExt)
	kubectlPath = filepath.Join(binDir, kubectlBinary+execExt)
	kubeadmPath = filepath.Join(binDir, kubeadmBinary+execExt)

	kubeBinariesPaths = []string{
		kubeletPath,
		kubectlPath,
		kubeadmPath,
	}
}
