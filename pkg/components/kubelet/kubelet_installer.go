package kubelet

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
	"go.goms.io/aks/AKSFlexNode/pkg/utils"
)

// Installer handles kubelet installation and configuration
type Installer struct {
	config *config.Config
	logger *logrus.Logger
}

// NewInstaller creates a new kubelet Installer
func NewInstaller(logger *logrus.Logger) *Installer {
	return &Installer{
		config: config.GetConfig(),
		logger: logger,
	}
}

// GetName returns the step name for the executor interface
func (i *Installer) GetName() string {
	return "KubeletInstaller"
}

// Execute installs and configures kubelet service
func (i *Installer) Execute(ctx context.Context) error {
	i.logger.Info("Installing and configuring kubelet")

	// Configure kubelet service with systemd unit file and default settings
	if err := i.configure(ctx); err != nil {
		return fmt.Errorf("failed to configure kubelet: %w", err)
	}

	i.logger.Info("Kubelet installed and configured successfully")
	return nil
}

// IsCompleted checks if kubelet service has been installed and configured
func (i *Installer) IsCompleted(ctx context.Context) bool {
	// Check if all required files exist
	if !utils.FileExists(KubeletDefaultsPath) {
		return false
	}
	if !utils.FileExists(KubeletServicePath) {
		return false
	}

	// Validate the configuration files have expected content
	if !i.validateKubeletConfiguration() {
		return false
	}

	// Check if kubelet service is running and healthy
	return i.isKubeletServiceHealthy()
}

// Validate validates prerequisites for kubelet installation
func (i *Installer) Validate(_ context.Context) error {
	i.logger.Debug("Validating prerequisites for kubelet installation")
	// No specific prerequisites for kubelet configuration
	return nil
}

// configure configures kubelet service with systemd unit file and default settings
func (i *Installer) configure(ctx context.Context) error {
	i.logger.Info("Configuring kubelet")

	// Clean up any existing corrupted configuration files
	if err := i.cleanupExistingConfiguration(); err != nil {
		i.logger.Warnf("Failed to cleanup existing kubelet configuration: %v", err)
		// Continue anyway - we'll overwrite the files
	}

	// Create kubelet defaults file
	if err := i.createKubeletDefaultsFile(); err != nil {
		return err
	}

	// Create kubelet containerd configuration
	if err := i.createKubeletContainerdConfig(); err != nil {
		return err
	}

	// Create main kubelet service
	if err := i.createKubeletServiceFile(); err != nil {
		return err
	}

	// Reload systemd to pick up the new kubelet configuration files
	i.logger.Info("Reloading systemd to pick up kubelet configuration changes")
	if err := utils.ReloadSystemd(); err != nil {
		return fmt.Errorf("failed to reload systemd after kubelet configuration: %w", err)
	}

	return nil
}

// cleanupExistingConfiguration removes any existing kubelet configuration that may be corrupted
func (i *Installer) cleanupExistingConfiguration() error {
	i.logger.Debug("Cleaning up existing kubelet configuration files")

	// List of files to clean up
	filesToClean := []string{
		KubeletDefaultsPath,
		KubeletServicePath,
		KubeletContainerdConfig,
	}

	for _, file := range filesToClean {
		if utils.FileExists(file) {
			i.logger.Debugf("Removing existing kubelet config file: %s", file)
			if err := utils.RunSystemCommand("rm", "-f", file); err != nil {
				i.logger.Warnf("Failed to remove %s: %v", file, err)
			}
		}
	}

	return nil
}

// validateKubeletConfiguration validates that kubelet configuration files have expected content
func (i *Installer) validateKubeletConfiguration() bool {
	// Validate kubelet defaults file
	if !i.validateKubeletDefaultsFile() {
		return false
	}

	// Validate kubelet service file
	if !i.validateKubeletServiceFile() {
		return false
	}

	return true
}

// validateKubeletDefaultsFile checks if the kubelet defaults file has expected content
func (i *Installer) validateKubeletDefaultsFile() bool {
	output, err := utils.RunCommandWithOutput("cat", KubeletDefaultsPath)
	if err != nil {
		i.logger.Debugf("Failed to read kubelet defaults file: %v", err)
		return false
	}

	// Check for key configuration markers
	expectedSettings := []string{
		"KUBELET_NODE_LABELS=",
		"KUBELET_CONFIG_FILE_FLAGS=",
		"KUBELET_TLS_BOOTSTRAP_FLAGS=",
		"KUBELET_FLAGS=",
		"--cgroup-driver=systemd",
		"--authorization-mode=Webhook",
	}

	for _, setting := range expectedSettings {
		if !strings.Contains(output, setting) {
			i.logger.Debugf("Missing expected setting in kubelet defaults file: %s", setting)
			return false
		}
	}

	return true
}

// validateKubeletServiceFile checks if the kubelet service file has expected content
func (i *Installer) validateKubeletServiceFile() bool {
	output, err := utils.RunCommandWithOutput("cat", KubeletServicePath)
	if err != nil {
		i.logger.Debugf("Failed to read kubelet service file: %v", err)
		return false
	}

	// Check for key configuration markers
	expectedSettings := []string{
		"[Unit]",
		"Description=Kubelet",
		"ExecStart=/usr/local/bin/kubelet",
		"WantedBy=multi-user.target",
	}

	for _, setting := range expectedSettings {
		if !strings.Contains(output, setting) {
			i.logger.Debugf("Missing expected setting in kubelet service file: %s", setting)
			return false
		}
	}

	return true
}

// isKubeletServiceHealthy checks if the kubelet service is running and healthy
func (i *Installer) isKubeletServiceHealthy() bool {
	// Check if kubelet service is active (running)
	if err := utils.RunSystemCommand("systemctl", "is-active", "--quiet", "kubelet"); err != nil {
		i.logger.Debugf("Kubelet service is not active: %v", err)
		return false
	}

	// Check if kubelet service is enabled
	if err := utils.RunSystemCommand("systemctl", "is-enabled", "--quiet", "kubelet"); err != nil {
		i.logger.Debugf("Kubelet service is not enabled: %v", err)
		return false
	}

	i.logger.Debug("Kubelet service is running and healthy")
	return true
}

// createKubeletDefaultsFile creates the kubelet defaults configuration file
func (i *Installer) createKubeletDefaultsFile() error {
	// Create kubelet default config
	labels := make([]string, 0, len(i.config.Node.Labels))
	for key, value := range i.config.Node.Labels {
		labels = append(labels, fmt.Sprintf("%s=%s", key, value))
	}

	kubeletDefaults := fmt.Sprintf(`KUBELET_NODE_LABELS="%s"
KUBELET_CONFIG_FILE_FLAGS="--kubeconfig=/etc/kubernetes/admin.conf"
KUBELET_TLS_BOOTSTRAP_FLAGS=""
KUBELET_FLAGS="\
  --address=0.0.0.0 \
  --anonymous-auth=false \
  --authentication-token-webhook=true \
  --authorization-mode=Webhook \
  --cgroup-driver=systemd \
  --cgroups-per-qos=true \
  --enforce-node-allocatable=pods \
  --event-qps=0  \
  --eviction-hard=%s  \
  --kube-reserved=%s  \
  --image-gc-high-threshold=%d  \
  --image-gc-low-threshold=%d  \
  --max-pods=%d  \
  --node-status-update-frequency=10s  \
  --pod-infra-container-image=%s  \
  --pod-max-pids=-1  \
  --protect-kernel-defaults=true  \
  --read-only-port=0  \
  --resolv-conf=/run/systemd/resolve/resolv.conf  \
  --streaming-connection-idle-timeout=4h  \
  --tls-cipher-suites=TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_RSA_WITH_AES_256_GCM_SHA384,TLS_RSA_WITH_AES_128_GCM_SHA256 \
  "`,
		strings.Join(labels, ","),
		utils.MapToEvictionThresholds(i.config.Node.Kubelet.EvictionHard, ","),
		utils.MapToKeyValuePairs(i.config.Node.Kubelet.KubeReserved, ","),
		i.config.Node.Kubelet.ImageGCHighThreshold,
		i.config.Node.Kubelet.ImageGCLowThreshold,
		i.config.Node.MaxPods,
		i.config.Containerd.PauseImage)

	// Ensure /etc/default directory exists
	if err := utils.RunSystemCommand("mkdir", "-p", EtcDefaultDir); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", EtcDefaultDir, err)
	}

	// Write kubelet defaults file atomically with proper permissions
	if err := utils.WriteFileAtomicSystem(KubeletDefaultsPath, []byte(kubeletDefaults), 0644); err != nil {
		return fmt.Errorf("failed to create kubelet defaults file: %w", err)
	}

	return nil
}

// createKubeletContainerdConfig creates the kubelet containerd configuration
func (i *Installer) createKubeletContainerdConfig() error {
	containerdConf := `[Service]
Environment=KUBELET_CONTAINERD_FLAGS="--runtime-request-timeout=15m --container-runtime-endpoint=unix:///run/containerd/containerd.sock"`

	// Ensure kubelet service.d directory exists
	if err := utils.RunSystemCommand("mkdir", "-p", KubeletServiceDir); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", KubeletServiceDir, err)
	}

	// Write kubelet containerd config file atomically with proper permissions
	if err := utils.WriteFileAtomicSystem(KubeletContainerdConfig, []byte(containerdConf), 0644); err != nil {
		return fmt.Errorf("failed to create kubelet containerd config file: %w", err)
	}

	return nil
}

// createKubeletServiceFile creates the main kubelet systemd service file
func (i *Installer) createKubeletServiceFile() error {
	kubeletService := `[Unit]
Description=Kubelet
ConditionPathExists=/usr/local/bin/kubelet
[Service]
Restart=always
EnvironmentFile=/etc/default/kubelet
SuccessExitStatus=143
# Ace does not recall why this is done
ExecStartPre=/bin/bash -c "if [ $(mount | grep \"/var/lib/kubelet\" | wc -l) -le 0 ] ; then /bin/mount --bind /var/lib/kubelet /var/lib/kubelet ; fi"
ExecStartPre=/bin/mount --make-shared /var/lib/kubelet
ExecStartPre=-/sbin/ebtables -t nat --list
ExecStartPre=-/sbin/iptables -t nat --numeric --list
ExecStart=/usr/local/bin/kubelet \
        --enable-server \
        --node-labels="${KUBELET_NODE_LABELS}" \
        --v=2 \
        --volume-plugin-dir=/etc/kubernetes/volumeplugins \
        --pod-manifest-path=/etc/kubernetes/manifests/ \
        $KUBELET_TLS_BOOTSTRAP_FLAGS \
        $KUBELET_CONFIG_FILE_FLAGS \
        $KUBELET_CONTAINERD_FLAGS \
        $KUBELET_FLAGS
[Install]
WantedBy=multi-user.target`

	// Write kubelet service file atomically with proper permissions
	if err := utils.WriteFileAtomicSystem(KubeletServicePath, []byte(kubeletService), 0644); err != nil {
		return fmt.Errorf("failed to create kubelet service file: %w", err)
	}

	return nil
}
