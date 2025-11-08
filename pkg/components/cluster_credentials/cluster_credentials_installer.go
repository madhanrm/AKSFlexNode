package cluster_credentials

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/auth"
	"go.goms.io/aks/AKSFlexNode/pkg/azure"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
	"go.goms.io/aks/AKSFlexNode/pkg/utils"
)

// Installer handles downloading AKS cluster credentials
type Installer struct {
	config       *config.Config
	logger       *logrus.Logger
	authProvider *auth.AuthProvider
}

// NewInstaller creates a new cluster credentials Installer
func NewInstaller(logger *logrus.Logger) *Installer {
	return &Installer{
		config:       config.GetConfig(),
		logger:       logger,
		authProvider: auth.NewAuthProvider(),
	}
}

// GetName returns the step name
func (i *Installer) GetName() string {
	return "ClusterCredentialsDownloaded"
}

// Validate validates prerequisites for downloading cluster credentials
func (i *Installer) Validate(ctx context.Context) error {
	return nil
}

// Execute downloads the AKS cluster credentials and configures kubectl
func (i *Installer) Execute(ctx context.Context) error {
	i.logger.Info("Downloading AKS cluster credentials using Azure Arc managed identity")

	// Get management token using ARC managed identity with retry
	i.logger.Debug("Acquiring managed identity credential...")
	cred, err := i.authProvider.ArcCredential()
	if err != nil {
		return fmt.Errorf("failed to get managed identity credential (ensure Azure Arc agent is running and properly configured): %w", err)
	}

	i.logger.Infof("Successfully acquired managed identity credential")

	// Fetch cluster credentials from Azure using SDK
	i.logger.Infof("Fetching cluster credentials for %s in resource group %s",
		i.config.Azure.TargetCluster.Name, i.config.Azure.TargetCluster.ResourceGroup)

	kubeconfigData, err := azure.GetClusterCredentials(ctx, cred, i.logger)
	if err != nil {
		return fmt.Errorf("failed to fetch cluster credentials from Azure: %w", err)
	}

	if len(kubeconfigData) == 0 {
		return fmt.Errorf("received empty kubeconfig data from Azure")
	}

	i.logger.Infof("Successfully retrieved cluster credentials (%d bytes)", len(kubeconfigData))

	// Save kubeconfig to file with enhanced error handling
	if err := i.saveKubeconfigFile(kubeconfigData); err != nil {
		return fmt.Errorf("failed to save cluster credentials: %w", err)
	}

	i.logger.Infof("Cluster credentials downloaded and saved successfully")
	return nil
}

// IsCompleted checks if cluster credentials have been downloaded and kubeconfig is available
func (i *Installer) IsCompleted(ctx context.Context) bool {
	adminKubeconfigPath := filepath.Join(i.config.Paths.Kubernetes.ConfigDir, "admin.conf")
	return utils.FileExists(adminKubeconfigPath)
}

// saveKubeconfigFile saves the kubeconfig data to the admin.conf file
func (i *Installer) saveKubeconfigFile(kubeconfigData []byte) error {
	kubeconfigPath := filepath.Join(i.config.Paths.Kubernetes.ConfigDir, "admin.conf")

	// Ensure the kubernetes config directory exists
	if err := utils.RunSystemCommand("mkdir", "-p", i.config.Paths.Kubernetes.ConfigDir); err != nil {
		return fmt.Errorf("failed to create kubernetes config directory: %w", err)
	}

	// Write kubeconfig using a temporary file and sudo to handle permissions
	tempFile, err := utils.CreateTempFile("kubeconfig-*.conf", kubeconfigData)
	if err != nil {
		return fmt.Errorf("failed to create temporary kubeconfig file: %w", err)
	}
	defer utils.CleanupTempFile(tempFile.Name())
	defer tempFile.Close()

	// Copy the temporary file to the final location with proper permissions
	if err := utils.RunSystemCommand("cp", tempFile.Name(), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to copy kubeconfig to final location: %w", err)
	}

	// Set proper ownership and permissions
	if err := utils.RunSystemCommand("chmod", "600", kubeconfigPath); err != nil {
		return fmt.Errorf("failed to set kubeconfig permissions: %w", err)
	}

	return nil
}
