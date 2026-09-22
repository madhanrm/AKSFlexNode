//go:build windows
// +build windows

package kubelet

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v5"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"go.goms.io/aks/AKSFlexNode/pkg/auth"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
	"go.goms.io/aks/AKSFlexNode/pkg/platform"
	"go.goms.io/aks/AKSFlexNode/pkg/utils"
)

// Installer handles kubelet installation and configuration on Windows
type Installer struct {
	config   *config.Config
	logger   *logrus.Logger
	platform platform.Platform
	mcClient *armcontainerservice.ManagedClustersClient
}

// NewInstaller creates a new kubelet Installer for Windows
func NewInstaller(logger *logrus.Logger) *Installer {
	return &Installer{
		config:   config.GetConfig(),
		logger:   logger,
		platform: platform.Current(),
	}
}

// GetName returns the step name for the executor interface
func (i *Installer) GetName() string {
	return "KubeletInstaller"
}

// Execute installs and configures kubelet service on Windows
func (i *Installer) Execute(ctx context.Context) error {
	i.logger.Info("Installing and configuring kubelet for Windows")

	// Set up mc client for getting cluster info
	if err := i.setUpClients(); err != nil {
		return fmt.Errorf("failed to set up Azure SDK clients: %w", err)
	}

	// Configure kubelet
	if err := i.configure(ctx); err != nil {
		return fmt.Errorf("failed to configure kubelet: %w", err)
	}

	i.logger.Info("Kubelet installed and configured successfully")
	return nil
}

// IsCompleted checks if kubelet service has been installed and configured
func (i *Installer) IsCompleted(ctx context.Context) bool {
	// Enforce reconfiguration every time to ensure latest settings
	return false
}

// Validate validates prerequisites for kubelet installation
func (i *Installer) Validate(_ context.Context) error {
	i.logger.Debug("Validating prerequisites for kubelet installation")

	// Check if kubelet binary exists
	kubeletPath := filepath.Join(i.platform.Paths().KubeletBinDir, "kubelet.exe")
	if _, err := os.Stat(kubeletPath); os.IsNotExist(err) {
		return fmt.Errorf("kubelet binary not found at %s - run kube_binaries step first", kubeletPath)
	}

	return nil
}

// configure configures kubelet for Windows
func (i *Installer) configure(ctx context.Context) error {
	i.logger.Info("Configuring kubelet for Windows")

	// Step 1: Create required directories
	if err := i.createRequiredDirectories(); err != nil {
		return fmt.Errorf("failed to create required directories: %w", err)
	}

	// Step 2: Create Arc token script for exec credential authentication
	if err := i.createArcTokenScript(); err != nil {
		return fmt.Errorf("failed to create Arc token script: %w", err)
	}

	// Step 3: Create kubeconfig with exec credential provider
	if err := i.createKubeconfigWithExecCredential(ctx); err != nil {
		return fmt.Errorf("failed to create kubeconfig: %w", err)
	}

	// Step 4: Register kubelet as Windows service
	if err := i.registerKubeletService(); err != nil {
		return fmt.Errorf("failed to register kubelet service: %w", err)
	}

	return nil
}

// createRequiredDirectories creates directories that kubelet expects to exist
func (i *Installer) createRequiredDirectories() error {
	i.logger.Info("Creating required directories for kubelet")

	for _, dir := range kubeletDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			i.logger.Debugf("Creating directory: %s", dir)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}

	i.logger.Info("Required directories created successfully")
	return nil
}

// createArcTokenScript creates the Arc token script for exec credential authentication on Windows
func (i *Installer) createArcTokenScript() error {
	i.logger.Info("Creating Arc token script for Windows")

	// PowerShell script to get Arc HIMDS token and output ExecCredential format
	tokenScript := fmt.Sprintf(`# Arc HIMDS token script for kubelet exec credential authentication
# This script fetches an AAD token from Azure Arc HIMDS and outputs it in ExecCredential format

$ErrorActionPreference = "Stop"

# Azure Arc HIMDS endpoint
$apiVersion = "2020-06-01"
$resource = "%s"  # AKS service resource ID
$endpoint = "http://localhost:40342/metadata/identity/oauth2/token?api-version=$apiVersion&resource=$resource"

try {
    # First request to get the challenge
    $response = $null
    try {
        $response = Invoke-WebRequest -Uri $endpoint -Headers @{Metadata='True'} -UseBasicParsing -ErrorAction Stop
    } catch {
        # Get the WWW-Authenticate header for the secret file path
        $wwwAuthHeader = $_.Exception.Response.Headers | Where-Object { $_.Key -eq "WWW-Authenticate" } | Select-Object -ExpandProperty Value
        if (-not $wwwAuthHeader) {
            # Try alternative method to get header
            $wwwAuthHeader = $_.Exception.Response.Headers.GetValues("WWW-Authenticate")
        }
        
        if ($wwwAuthHeader -match "Basic realm=(.+)") {
            $secretFilePath = $matches[1].Trim('"')
        } else {
            throw "Could not find secret file path in WWW-Authenticate header"
        }
        
        # Read the challenge token from the file
        $secret = Get-Content -Path $secretFilePath -Raw -ErrorAction Stop
        $secret = $secret.Trim()
        
        # Make the authenticated request
        $response = Invoke-WebRequest -Uri $endpoint -Headers @{Metadata='True'; Authorization="Basic $secret"} -UseBasicParsing -ErrorAction Stop
    }
    
    # Parse the token response
    $tokenResponse = $response.Content | ConvertFrom-Json
    $accessToken = $tokenResponse.access_token
    $expiresOn = $tokenResponse.expires_on
    
    # Convert expires_on (Unix timestamp) to ISO 8601 format
    $expirationTime = [DateTimeOffset]::FromUnixTimeSeconds([long]$expiresOn).ToString("yyyy-MM-ddTHH:mm:ssZ")
    
    # Output in ExecCredential format
    $execCredential = @{
        kind = "ExecCredential"
        apiVersion = "client.authentication.k8s.io/v1beta1"
        spec = @{
            interactive = $false
        }
        status = @{
            expirationTimestamp = $expirationTime
            token = $accessToken
        }
    }
    
    # Output as JSON
    $execCredential | ConvertTo-Json -Depth 10
    
} catch {
    Write-Error "Failed to get Arc token: $_"
    exit 1
}
`, aksServiceResourceID)

	// Write the token script
	if err := os.WriteFile(kubeletTokenScriptPath, []byte(tokenScript), 0755); err != nil {
		return fmt.Errorf("failed to write Arc token script: %w", err)
	}

	i.logger.Infof("Arc token script created at %s", kubeletTokenScriptPath)
	return nil
}

// createKubeconfigWithExecCredential creates kubeconfig with exec credential provider for Arc authentication
func (i *Installer) createKubeconfigWithExecCredential(ctx context.Context) error {
	i.logger.Info("Creating kubeconfig with exec credential provider")

	kubeconfig, err := i.getClusterCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cluster credentials: %w", err)
	}

	serverURL, caCertData, err := utils.ExtractClusterInfo(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to extract cluster info from kubeconfig: %w", err)
	}

	// Create cluster configuration - CA cert is required for secure connections
	var clusterConfig string
	if caCertData != "" {
		clusterConfig = fmt.Sprintf(`- cluster:
    certificate-authority-data: %s
    server: %s
  name: %s`, caCertData, serverURL, i.config.Azure.TargetCluster.Name)
	} else {
		// CA certificate is required for secure cluster communication
		// Falling back to insecure connections exposes the cluster to MITM attacks
		return fmt.Errorf("CA certificate data is required but not available from cluster credentials; cannot configure secure kubelet connection")
	}

	// Escape backslashes for the token script path in YAML
	tokenScriptPathEscaped := strings.ReplaceAll(kubeletTokenScriptPath, "\\", "\\\\")

	// Create kubeconfig with exec credential provider pointing to PowerShell token script
	kubeconfigContent := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
%s
contexts:
- context:
    cluster: %s
    user: arc-user
  name: arc-context
current-context: arc-context
users:
- name: arc-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: powershell.exe
      args:
      - -ExecutionPolicy
      - Bypass
      - -File
      - %s
      env: null
      provideClusterInfo: false
`,
		clusterConfig,
		i.config.Azure.TargetCluster.Name,
		tokenScriptPathEscaped)

	// Write kubeconfig file
	if err := os.WriteFile(kubeletKubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		return fmt.Errorf("failed to create kubeconfig file: %w", err)
	}

	i.logger.Infof("Kubeconfig created at %s", kubeletKubeconfigPath)
	return nil
}

// registerKubeletService registers kubelet as a Windows service
func (i *Installer) registerKubeletService() error {
	i.logger.Info("Registering kubelet as Windows service")

	// Build kubelet arguments
	kubeletPath := filepath.Join(i.platform.Paths().KubeletBinDir, "kubelet.exe")
	
	// Build node labels
	labels := make([]string, 0, len(i.config.Node.Labels))
	for key, value := range i.config.Node.Labels {
		labels = append(labels, fmt.Sprintf("%s=%s", key, value))
	}

	// Kubelet arguments for Windows (aligned with AKS Arc patterns)
	kubeletArgs := []string{
		"--enable-server",
		fmt.Sprintf("--kubeconfig=%s", kubeletKubeconfigPath),
		fmt.Sprintf("--pod-infra-container-image=%s", i.config.Containerd.PauseImage),
		fmt.Sprintf("--v=%d", i.config.Node.Kubelet.Verbosity),
		"--address=0.0.0.0",
		"--anonymous-auth=false",
		"--authentication-token-webhook=true",
		"--authorization-mode=Webhook",
		"--client-ca-file=",  // Will be populated by TLS bootstrap
		"--cluster-dns=10.0.0.10",  // Default AKS DNS
		"--cluster-domain=cluster.local",
		fmt.Sprintf("--cni-bin-dir=%s", i.platform.Paths().CNIBinDir),
		fmt.Sprintf("--cni-conf-dir=%s", i.platform.Paths().CNIConfDir),
		"--container-runtime-endpoint=npipe:////./pipe/containerd-containerd",
		"--event-qps=0",
		fmt.Sprintf("--eviction-hard=%s", mapToEvictionThresholds(i.config.Node.Kubelet.EvictionHard, ",")),
		fmt.Sprintf("--image-gc-high-threshold=%d", i.config.Node.Kubelet.ImageGCHighThreshold),
		fmt.Sprintf("--image-gc-low-threshold=%d", i.config.Node.Kubelet.ImageGCLowThreshold),
		fmt.Sprintf("--kube-reserved=%s", mapToKeyValuePairs(i.config.Node.Kubelet.KubeReserved, ",")),
		fmt.Sprintf("--max-pods=%d", i.config.Node.MaxPods),
		"--network-plugin=cni",
		"--node-status-update-frequency=10s",
		fmt.Sprintf("--pod-manifest-path=%s", kubeletManifestsDir),
		"--protect-kernel-defaults=false",  // Windows doesn't support this
		"--read-only-port=0",
		"--resolv-conf=",  // Windows uses system DNS
		"--streaming-connection-idle-timeout=4h",
		fmt.Sprintf("--volume-plugin-dir=%s", kubeletVolumePluginDir),
	}

	// Add node labels if configured
	if len(labels) > 0 {
		kubeletArgs = append(kubeletArgs, fmt.Sprintf("--node-labels=%s", strings.Join(labels, ",")))
	}

	// Create the service using platform service manager
	serviceConfig := &platform.ServiceConfig{
		Name:          kubeletServiceName,
		DisplayName:   "Kubernetes Kubelet",
		Description:   "Kubernetes node agent that manages pods and containers",
		BinaryPath:    kubeletPath,
		Args:          kubeletArgs,
		RestartPolicy: platform.RestartAlways,
		Dependencies:  []string{"containerd"},
	}

	if err := i.platform.Service().Install(serviceConfig); err != nil {
		// If service already exists, try to update it
		i.logger.Warnf("Failed to install kubelet service (may already exist): %v", err)
	}

	// Enable the service to start on boot
	if err := i.platform.Service().Enable(kubeletServiceName); err != nil {
		i.logger.Warnf("Failed to enable kubelet service: %v", err)
	}

	i.logger.Info("Kubelet service registered successfully")
	return nil
}

func (i *Installer) setUpClients() error {
	cred, err := auth.NewAuthProvider().UserCredential(config.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to get authentication credential: %w", err)
	}
	clusterSubID := i.config.GetTargetClusterSubscriptionID()
	clientFactory, err := armcontainerservice.NewClientFactory(clusterSubID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create Azure Container Service client factory: %w", err)
	}
	i.mcClient = clientFactory.NewManagedClustersClient()
	return nil
}

// getClusterCredentials retrieves cluster kube admin credentials using Azure SDK
func (i *Installer) getClusterCredentials(ctx context.Context) ([]byte, error) {
	cfg := config.GetConfig()
	clusterResourceGroup := cfg.GetTargetClusterResourceGroup()
	clusterName := cfg.GetTargetClusterName()
	i.logger.Infof("Fetching cluster credentials for cluster %s in resource group %s using Azure SDK",
		clusterName, clusterResourceGroup)

	// Get cluster admin credentials using the Azure SDK
	resp, err := i.mcClient.ListClusterAdminCredentials(ctx, clusterResourceGroup, clusterName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster admin credentials for %s in resource group %s: %w", clusterName, clusterResourceGroup, err)
	}

	if len(resp.Kubeconfigs) == 0 {
		return nil, fmt.Errorf("no kubeconfig found in cluster admin credentials response")
	}

	kubeconfig := resp.Kubeconfigs[0]
	if kubeconfig == nil {
		return nil, fmt.Errorf("kubeconfig is nil in the response")
	}

	i.logger.Debugf("Found %d kubeconfig(s), using the first one of name %s", len(resp.Kubeconfigs), to.String(kubeconfig.Name))

	if len(kubeconfig.Value) == 0 {
		return nil, fmt.Errorf("kubeconfig value is empty")
	}

	return kubeconfig.Value, nil
}

// mapToKeyValuePairs converts a map to key=value pairs joined by separator
func mapToKeyValuePairs(m map[string]string, separator string) string {
	pairs := make([]string, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, separator)
}

// mapToEvictionThresholds converts a map to key<value pairs for kubelet eviction thresholds
func mapToEvictionThresholds(m map[string]string, separator string) string {
	pairs := make([]string, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, fmt.Sprintf("%s<%s", k, v))
	}
	return strings.Join(pairs, separator)
}
