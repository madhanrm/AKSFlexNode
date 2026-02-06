//go:build windows
// +build windows

package cni

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
	"go.goms.io/aks/AKSFlexNode/pkg/platform"
)

// Installer handles Calico CNI setup and installation operations on Windows
type Installer struct {
	config   *config.Config
	logger   *logrus.Logger
	platform platform.Platform
}

// NewInstaller creates a new CNI setup Installer for Windows
func NewInstaller(logger *logrus.Logger) *Installer {
	return &Installer{
		config:   config.GetConfig(),
		logger:   logger,
		platform: platform.Current(),
	}
}

// GetName returns the step name
func (i *Installer) GetName() string {
	return "CNISetup"
}

// Validate validates prerequisites for Calico CNI setup on Windows
func (i *Installer) Validate(ctx context.Context) error {
	// Validate Calico version format
	calicoVersion := getCalicoVersion(i.config)
	if calicoVersion == "" {
		return fmt.Errorf("Calico version cannot be empty")
	}

	// Check if containerd is installed (required for Calico)
	containerdPath := filepath.Join(i.platform.Paths().ContainerdBinDir, "containerd.exe")
	if _, err := os.Stat(containerdPath); os.IsNotExist(err) {
		return fmt.Errorf("containerd must be installed before CNI setup")
	}

	return nil
}

// Execute configures Calico CNI for Windows
func (i *Installer) Execute(ctx context.Context) error {
	i.logger.Info("Setting up Calico CNI for Windows")

	// Step 1: Prepare CNI directories
	i.logger.Info("Step 1: Preparing CNI directories")
	if err := i.prepareCNIDirectories(); err != nil {
		return fmt.Errorf("failed to prepare CNI directories: %w", err)
	}
	i.logger.Info("CNI directories are ready")

	// Step 2: Download and install Calico for Windows
	i.logger.Info("Step 2: Installing Calico for Windows")
	if err := i.installCalico(); err != nil {
		i.logger.Errorf("Calico installation failed: %v", err)
		return fmt.Errorf("failed to install Calico version %s: %w", getCalicoVersion(i.config), err)
	}
	i.logger.Info("Calico installed successfully")

	// Step 3: Create Calico config.ps1 (AKS Arc pattern)
	i.logger.Info("Step 3: Creating Calico config.ps1")
	if err := i.createCalicoConfigPS1(); err != nil {
		i.logger.Errorf("Calico config.ps1 creation failed: %v", err)
		return fmt.Errorf("failed to create Calico config.ps1: %w", err)
	}
	i.logger.Info("Calico config.ps1 created successfully")

	// Step 4: Create Calico CNI configuration
	i.logger.Info("Step 4: Creating Calico CNI configuration")
	if err := i.createCalicoConfig(); err != nil {
		i.logger.Errorf("Calico CNI configuration failed: %v", err)
		return fmt.Errorf("failed to create Calico CNI config: %w", err)
	}
	i.logger.Info("Calico CNI configuration created successfully")

	// Step 5: Configure HNS network (for VXLAN overlay)
	i.logger.Info("Step 5: Configuring HNS network")
	if err := i.configureHNSNetwork(); err != nil {
		i.logger.Warnf("HNS network configuration failed (may be configured later): %v", err)
		// Don't fail - HNS network may be configured by Calico service on startup
	}

	i.logger.Info("Calico CNI setup completed successfully")
	return nil
}

// IsCompleted checks if Calico CNI configuration has been set up properly
func (i *Installer) IsCompleted(ctx context.Context) bool {
	// Validate Step 1: CNI directories
	for _, dir := range cniDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			i.logger.Debugf("CNI directory not found: %s", dir)
			return false
		}
	}

	// Validate Step 2: Calico CNI plugin binaries
	for _, plugin := range requiredCNIPlugins {
		pluginPath := filepath.Join(DefaultCNIBinDir, plugin)
		if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
			i.logger.Debugf("CNI plugin not found: %s", plugin)
			return false
		}
	}

	// Validate Step 3: Calico CNI configuration
	configPath := filepath.Join(DefaultCNIConfDir, calicoConfigFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		i.logger.Debug("Calico CNI configuration file not found")
		return false
	}

	i.logger.Debug("Calico CNI setup validation passed - all components properly configured")
	return true
}

func (i *Installer) prepareCNIDirectories() error {
	for _, dir := range cniDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			i.logger.Debugf("Creating CNI directory: %s", dir)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create CNI directory %s: %w", dir, err)
			}
		}
	}
	return nil
}

// installCalico downloads and installs Calico for Windows
func (i *Installer) installCalico() error {
	// Check if already installed
	if i.canSkipCalicoInstallation() {
		i.logger.Info("Calico plugins are already installed, skipping download")
		return nil
	}

	calicoVersion := getCalicoVersion(i.config)
	
	// Try primary URL first (Azure CDN), then fallback to GitHub
	downloadURLs := []string{
		fmt.Sprintf(calicoWindowsZipURL, calicoVersion, calicoVersion),
		fmt.Sprintf(calicoGitHubZipURL, calicoVersion, calicoVersion),
	}

	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("calico-windows-v%s.zip", calicoVersion))
	var downloadErr error

	for _, downloadURL := range downloadURLs {
		i.logger.Infof("Downloading Calico for Windows v%s from: %s", calicoVersion, downloadURL)
		
		if err := i.downloadFile(downloadURL, tempFile); err != nil {
			i.logger.Warnf("Failed to download from %s: %v, trying next URL...", downloadURL, err)
			downloadErr = err
			continue
		}
		downloadErr = nil
		break
	}

	if downloadErr != nil {
		return fmt.Errorf("failed to download Calico from all sources: %w", downloadErr)
	}
	defer os.Remove(tempFile)

	// Extract to CalicoWindows directory
	i.logger.Infof("Extracting Calico to %s", CalicoDir)
	if err := i.extractZip(tempFile, CalicoDir); err != nil {
		return fmt.Errorf("failed to extract Calico: %w", err)
	}

	// Copy CNI plugins to CNI bin directory
	i.logger.Info("Copying CNI plugins to CNI bin directory")
	if err := i.copyCNIPlugins(); err != nil {
		return fmt.Errorf("failed to copy CNI plugins: %w", err)
	}

	return nil
}

func (i *Installer) canSkipCalicoInstallation() bool {
	for _, plugin := range requiredCNIPlugins {
		pluginPath := filepath.Join(DefaultCNIBinDir, plugin)
		if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func (i *Installer) downloadFile(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (i *Installer) extractZip(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		// Construct the destination path and clean it to prevent zip slip attacks
		destPath := filepath.Clean(filepath.Join(destDir, file.Name))
		cleanDestDir := filepath.Clean(destDir) + string(os.PathSeparator)

		// Check for zip slip vulnerability - ensure destPath is within destDir
		if !strings.HasPrefix(destPath, cleanDestDir) && destPath != filepath.Clean(destDir) {
			return fmt.Errorf("invalid file path in zip (potential zip slip attack): %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(destPath, file.Mode())
			continue
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
		}

		// Extract file
		if err := i.extractFile(file, destPath); err != nil {
			return err
		}
	}

	return nil
}

func (i *Installer) extractFile(file *zip.File, destPath string) error {
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open zip entry %s: %w", file.Name, err)
	}
	defer src.Close()

	dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func (i *Installer) copyCNIPlugins() error {
	// Calico for Windows puts CNI plugins in CalicoWindows\cni directory
	sourceDir := filepath.Join(CalicoDir, "cni")

	// List of CNI plugins to copy
	plugins := []string{
		"calico.exe",
		"calico-ipam.exe",
		"host-local.exe",
		"win-bridge.exe",
		"win-overlay.exe",
		"flannel.exe",
	}

	for _, plugin := range plugins {
		src := filepath.Join(sourceDir, plugin)
		dst := filepath.Join(DefaultCNIBinDir, plugin)

		// Check if source exists
		if _, err := os.Stat(src); os.IsNotExist(err) {
			i.logger.Debugf("Plugin %s not found in Calico package (may not be needed)", plugin)
			continue
		}

		// Copy file
		if err := i.copyFile(src, dst); err != nil {
			// Only fail for required plugins
			for _, required := range requiredCNIPlugins {
				if plugin == required {
					return fmt.Errorf("failed to copy required plugin %s: %w", plugin, err)
				}
			}
			i.logger.Warnf("Failed to copy optional plugin %s: %v", plugin, err)
		}
	}

	return nil
}

func (i *Installer) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// createCalicoConfig creates the Calico CNI configuration for Windows
func (i *Installer) createCalicoConfig() error {
	configPath := filepath.Join(DefaultCNIConfDir, calicoConfigFile)

	// Use default values - these will be updated by kubelet configuration step
	// or can be overridden via config file
	serviceCIDR := "10.0.0.0/16" // Default AKS service CIDR
	dnsServiceIP := "10.0.0.10"  // Default AKS DNS IP

	// Determine networking backend
	backend := VXLAN // Default to VXLAN overlay

	// Generate configuration from template
	configContent, err := i.generateCalicoConfig(backend, serviceCIDR, dnsServiceIP)
	if err != nil {
		return fmt.Errorf("failed to generate Calico config: %w", err)
	}

	// Write configuration file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write Calico config: %w", err)
	}

	i.logger.Infof("Calico CNI configuration written to %s", configPath)
	return nil
}

func (i *Installer) generateCalicoConfig(backend NetworkingBackend, serviceCIDR, dnsServiceIP string) (string, error) {
	tmpl := `{
  "name": "Calico",
  "cniVersion": "{{.CNIVersion}}",
  "plugins": [
    {
      "type": "calico",
      "mode": "{{.Mode}}",
      "vxlan_mac_prefix": "0E-2A",
      "vxlan_vni": 4096,
      "policy": {
        "type": "k8s"
      },
      "log_level": "Info",
      "windows_use_single_network": true,
      "capabilities": {
        "dns": true
      },
      "DNS": {
        "Nameservers": ["{{.DNSServiceIP}}"],
        "Search": [
          "svc.cluster.local"
        ]
      },
      "nodename_file": "{{.NodenameFile}}",
      "datastore_type": "kubernetes",
      "ipam": {
        "type": "calico-ipam",
        "subnet": "usePodCidr"
      },
      "kubernetes": {
        "kubeconfig": "{{.Kubeconfig}}"
      }
    }
  ]
}`

	data := struct {
		CNIVersion   string
		Mode         string
		DNSServiceIP string
		NodenameFile string
		Kubeconfig   string
	}{
		CNIVersion:   DefaultCNISpecVersion,
		Mode:         string(backend),
		DNSServiceIP: dnsServiceIP,
		NodenameFile: filepath.Join(CalicoDataDir, "nodename"),
		Kubeconfig:   filepath.Join(i.platform.Paths().KubeletConfigDir, "kubelet.kubeconfig"),
	}

	// Windows paths need to be escaped for JSON
	data.NodenameFile = strings.ReplaceAll(data.NodenameFile, "\\", "\\\\")
	data.Kubeconfig = strings.ReplaceAll(data.Kubeconfig, "\\", "\\\\")

	t, err := template.New("calico").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// configureHNSNetwork configures the HNS network for Calico
func (i *Installer) configureHNSNetwork() error {
	// HNS network is typically configured by the Calico service on startup
	// or via the install-calico.ps1 script from the Calico package

	// Check if we have the Calico install script
	installScript := filepath.Join(CalicoDir, "install-calico.ps1")
	if _, err := os.Stat(installScript); os.IsNotExist(err) {
		i.logger.Info("Calico install script not found, HNS network will be configured by Calico service")
		return nil
	}

	// The HNS network configuration is complex and involves:
	// 1. Creating an external vSwitch bound to the management NIC
	// 2. Creating a Calico network with VXLAN backend
	// 3. Configuring IP forwarding and required firewall rules

	// For now, we'll let the Calico service handle this on startup
	// The service will create the necessary HNS network when it starts
	i.logger.Info("HNS network will be configured by Calico service on startup")
	return nil
}

// createCalicoConfigPS1 creates the config.ps1 file for Calico Windows
// This follows the AKS Arc pattern from Calico-Windows repo
func (i *Installer) createCalicoConfigPS1() error {
	configPath := filepath.Join(CalicoDir, "config.ps1")

	// Use default values - these will be updated during kubelet configuration
	// or can be provided in config file
	serviceCIDR := "10.0.0.0/16"
	clusterCIDR := "10.244.0.0/16"
	dnsServiceIP := "10.0.0.10"

	// Generate config.ps1 content (aligned with AKS Arc Calico-Windows repo)
	configContent := fmt.Sprintf(`
## Cluster configuration:

# KUBE_NETWORK should be set to a regular expression that matches the HNS network(s) used for pods.
# The default, "Calico.*", is correct for Calico CNI. 
$env:KUBE_NETWORK = "Calico.*"

# Set this to one of the following values:
# - "vxlan" for Calico VXLAN networking
# - "none" to disable the Calico CNI plugin (so that you can use another plugin).
$env:CALICO_NETWORKING_BACKEND="vxlan"
$env:CNI_MTU = "1450"

# Set to match your Kubernetes service CIDR.
$env:K8S_SERVICE_CIDR = "%s"
$env:CALICO_IPV4POOL_CIDR = "%s"
$env:DNS_NAME_SERVERS = "%s"
$env:DNS_SEARCH = "svc.cluster.local"


## Datastore configuration:

# Set this to "kubernetes" to use the kubernetes datastore, or "etcdv3" for etcd.
$env:CALICO_DATASTORE_TYPE = "kubernetes"

# Set KUBECONFIG to the path of your kubeconfig file.
$env:KUBECONFIG = "c:\k\config"


## CNI configuration, only used for the "vxlan" networking backends.

# Place to install the CNI plugin to.  Should match kubelet's --cni-bin-dir.
$env:CNI_BIN_DIR = "c:\k\cni"
# Place to install the CNI config to.  Should be located in kubelet's --cni-conf-dir.
$env:CNI_CONF_DIR = "c:\k\cni\config"
$env:CNI_CONF_FILENAME = "10-calico.conf"
# IPAM type to use with Calico's CNI plugin.  One of "calico-ipam" or "host-local".
$env:CNI_IPAM_TYPE = "calico-ipam"

## VXLAN-specific configuration.

# The VXLAN VNI / VSID.  Must match the VXLANVNI felix configuration parameter used
# for Linux nodes.
$env:VXLAN_VNI = "4096"
# Prefix used when generating MAC addresses for virtual NICs.
$env:VXLAN_MAC_PREFIX = "0E-2A"


## Node configuration.

# The NODENAME variable should be set to match the Kubernetes Node name of this host.
# The default uses this node's hostname (which is the same as kubelet).
$env:NODENAME = $(hostname).ToLower()
# Similarly, CALICO_K8S_NODE_REF should be set to the Kubernetes Node name.
$env:CALICO_K8S_NODE_REF = $env:NODENAME

# The time out to wait for a valid IP of an interface to be assigned before initialising Calico
# after a reboot.
$env:STARTUP_VALID_IP_TIMEOUT = 90

# The IP of the node; the default will auto-detect a usable IP in most cases.
$env:IP = "autodetect"

## Logging.

$env:CALICO_LOG_DIR = "$PSScriptRoot\logs"

# Disable logging to file by default since the service wrapper will redirect our log to file.
$env:FELIX_LOGSEVERITYFILE = "none"
# Disable syslog logging, which is not supported on Windows.
$env:FELIX_LOGSEVERITYSYS = "none"

# NAT issue fix - Pattern should match network name like vEthernet (Ethernet 2)
$env:IP_AUTODETECTION_METHOD = "interface=vEthernet.*Ethernet.*"
`, serviceCIDR, clusterCIDR, dnsServiceIP)

	// Write the config file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write Calico config.ps1: %w", err)
	}

	i.logger.Infof("Calico config.ps1 written to %s", configPath)
	return nil
}

func getCalicoVersion(cfg *config.Config) string {
	if cfg.CNI.Version != "" {
		return cfg.CNI.Version
	}
	return DefaultCalicoVersion
}
