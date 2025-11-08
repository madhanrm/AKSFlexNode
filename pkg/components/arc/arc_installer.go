package arc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/hybridcompute/armhybridcompute"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"go.goms.io/aks/AKSFlexNode/pkg/utils"
)

// Installer handles Azure Arc installation operations
type Installer struct {
	*Base
}

// NewInstaller creates a new Arc installer
func NewInstaller(logger *logrus.Logger) *Installer {
	return &Installer{
		Base: NewBase(logger),
	}
}

// Validate validates prerequisites for Arc installation
func (i *Installer) Validate(ctx context.Context) error {
	// No specific prerequisites validation needed for Arc installation
	return nil
}

// GetName returns the step name
func (i *Installer) GetName() string {
	return "ArcInstall"
}

// Execute performs Arc setup as part of the bootstrap process
// This method is designed to be called from bootstrap steps and handles all Arc-related setup
// It stops on the first error to prevent partial setups
func (i *Installer) Execute(ctx context.Context) error {
	i.logger.Info("Starting Arc setup for bootstrap process")

	// Step 1: Install Arc agent
	i.logger.Info("Step 1: Installing Arc agent")
	if err := i.installArcAgent(ctx); err != nil {
		i.logger.Errorf("Failed to install Arc agent: %v", err)
		return fmt.Errorf("Arc bootstrap setup failed at agent installation: %w", err)
	}
	i.logger.Info("Successfully installed Arc agent")

	// Step 2: Register Arc machine with Azure
	i.logger.Info("Step 2: Registering Arc machine with Azure")
	machine, err := i.registerArcMachine(ctx)
	if err != nil {
		i.logger.Errorf("Failed to register Arc machine: %v", err)
		return fmt.Errorf("Arc bootstrap setup failed at machine registration: %w", err)
	}
	i.logger.Info("Successfully registered Arc machine with Azure")

	// Step 3: Assign RBAC roles to managed identity (if enabled)
	if i.config.GetArcAutoRoleAssignment() {
		i.logger.Info("Step 3: Assigning RBAC roles to managed identity")
		// wait a moment to ensure machine info is fully propagated
		time.Sleep(10 * time.Second)
		if err := i.assignRBACRoles(ctx, machine); err != nil {
			i.logger.Errorf("Failed to assign RBAC roles: %v", err)
			return fmt.Errorf("Arc bootstrap setup failed at RBAC role assignment: %w", err)
		}
		i.logger.Info("Successfully assigned RBAC roles")
	} else {
		i.logger.Info("Step 3: Skipping RBAC role assignment (autoRoleAssignment is disabled in config)")
	}

	// Step 4: Wait for permissions to become effective
	// Note: This step is needed regardless of autoRoleAssignment setting because:
	// - If autoRoleAssignment=true: we assigned roles and need to wait for them to be effective
	// - If autoRoleAssignment=false: customer must assign roles manually, and we still need to wait for them
	i.logger.Info("Step 4: Waiting for RBAC permissions to become effective")
	if err := i.waitForRBACPermissions(ctx, machine); err != nil {
		i.logger.Errorf("Failed while waiting for RBAC permissions: %v", err)
		return fmt.Errorf("Arc bootstrap setup failed while waiting for RBAC permissions: %w", err)
	}
	i.logger.Info("RBAC permissions are now effective")

	i.logger.Info("Arc setup for bootstrap completed successfully")
	return nil
}

// IsCompleted checks if Arc setup has been completed
// This can be used by bootstrap steps to verify completion status
func (i *Installer) IsCompleted(ctx context.Context) bool {
	i.logger.Debug("Checking Arc setup completion status")

	// Check if Arc agent is running
	if !isArcServicesRunning() {
		i.logger.Debug("Arc agent is not running")
		return false
	}

	// Check if machine is registered with Arc
	if _, err := i.GetArcMachine(ctx); err != nil {
		i.logger.Debugf("Arc machine not registered or not accessible: %v", err)
		return false
	}

	i.logger.Debug("Arc setup appears to be completed")
	return true
}

// installArcAgent installs the Azure Arc agent on the system
func (i *Installer) installArcAgent(ctx context.Context) error {
	i.logger.Info("Installing Azure Arc agent")

	// Check if Arc agent is already installed and working
	if isArcAgentInstalled() {
		i.logger.Info("Azure Arc agent is already installed")
		return nil
	}

	// Check for filesystem corruption: package installed but files missing
	if i.isArcPackageCorrupted() {
		i.logger.Warn("Arc agent package corruption detected - forcing reinstall")
		if err := i.forceReinstallArcAgent(ctx); err != nil {
			return fmt.Errorf("failed to reinstall corrupted Arc agent: %w", err)
		}
		return nil
	}

	// Download and prepare installation script
	if err := i.downloadArcAgentScript(ctx); err != nil {
		return fmt.Errorf("failed to download Arc agent script: %w", err)
	}

	// Clean up script after installation
	defer i.cleanupInstallationScript()

	// Install prerequisites and Arc agent
	if err := i.installPrerequisites(); err != nil {
		return fmt.Errorf("failed to install prerequisites: %w", err)
	}

	if err := i.runArcAgentInstallation(ctx); err != nil {
		return fmt.Errorf("failed to run Arc agent installation: %w", err)
	}

	i.logger.Info("Azure Arc agent verification successful")
	return nil
}

// downloadArcAgentScript downloads and prepares the Arc agent installation script
func (i *Installer) downloadArcAgentScript(ctx context.Context) error {
	// Use wget to download (more reliable than custom download function) - needs sudo for temp file access
	cmd := exec.CommandContext(ctx, "sudo", "wget", arcAgentScriptURL, "-O", arcAgentTmpScriptPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download Arc agent installation script: %w", err)
	}

	// Make script executable using sudo (since file was downloaded with sudo)
	cmd = exec.CommandContext(ctx, "sudo", "chmod", "755", arcAgentTmpScriptPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to make script executable: %w", err)
	}

	return nil
}

// cleanupInstallationScript removes the temporary installation script
func (i *Installer) cleanupInstallationScript() {
	utils.RunCleanupCommand("rm", "-f", arcAgentTmpScriptPath)
}

// runArcAgentInstallation executes the Arc agent installation script with proper verification
func (i *Installer) runArcAgentInstallation(ctx context.Context) error {
	i.logger.Info("Running Arc agent installation script...")

	// Run the installation script without parameters to install the agent
	cmd := exec.CommandContext(ctx, "sudo", "bash", arcAgentTmpScriptPath)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Arc agent installation script failed: %w", err)
	}

	// Verify installation was successful by checking if azcmagent is now available
	i.logger.Info("Verifying Arc agent binary is accessible...")
	if !isArcAgentInstalled() {
		i.logger.Info("Arc agent not found in PATH, checking common installation locations...")

		// Check common installation paths
		var foundPath string
		for _, path := range arcPaths {
			i.logger.Infof("Checking for Arc agent at: %s", path)
			if _, statErr := os.Stat(path); statErr == nil {
				i.logger.Infof("Found Arc agent at: %s", path)
				foundPath = path
				break
			} else {
				i.logger.Infof("Arc agent not found at %s: %v", path, statErr)
			}
		}

		if foundPath != "" {
			// Automatically create symlink to make azcmagent available in PATH
			i.logger.Infof("Creating symlink to make Arc agent available in PATH")
			if err := i.createArcAgentSymlink(foundPath); err != nil {
				return fmt.Errorf("Arc agent installed at %s but failed to create PATH symlink: %w", foundPath, err)
			}
		} else {
			return fmt.Errorf("Arc agent installation script completed but azcmagent binary is not available in PATH or common locations (%v). The installation may have failed or been corrupted", arcPaths)
		}
	}

	i.logger.Info("Arc agent binary verification successful")
	return nil
}

// createArcAgentSymlink creates a symlink for azcmagent to make it available in PATH
func (i *Installer) createArcAgentSymlink(sourcePath string) error {
	i.logger.Infof("Arc agent found at %s, creating symlink to /usr/local/bin/azcmagent", sourcePath)
	cmd := exec.Command("sudo", "ln", "-sf", sourcePath, "/usr/local/bin/azcmagent")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Arc agent installed at %s but not in PATH. Failed to create symlink: %v. Please manually run: sudo ln -sf %s /usr/local/bin/azcmagent", sourcePath, err, sourcePath)
	}
	i.logger.Info("Successfully created symlink for azcmagent")
	return nil
}

// registerArcMachine registers the machine with Azure Arc using the Arc agent
func (i *Installer) registerArcMachine(ctx context.Context) (*armhybridcompute.Machine, error) {
	i.logger.Info("Registering machine with Azure Arc using Arc agent")

	// Check if already registered
	if machine, err := i.GetArcMachine(ctx); err == nil && machine != nil {
		i.logger.Infof("Machine already registered as Arc machine: %s", *machine.Name)
		return machine, nil
	}

	// Register using Arc agent command
	if err := i.runArcAgentConnect(ctx); err != nil {
		return nil, fmt.Errorf("failed to register Arc machine using agent: %w", err)
	}

	// Wait a moment for registration to complete
	i.logger.Info("Waiting for Arc machine registration to complete...")
	time.Sleep(10 * time.Second)

	// Verify registration by retrieving the machine
	machine, err := i.GetArcMachine(ctx)
	if err != nil {
		return nil, fmt.Errorf("Arc agent registration completed but failed to retrieve machine info: %w", err)
	}

	i.logger.Info("Arc machine registration completed successfully")
	return machine, nil
}

// runArcAgentConnect connects the machine to Azure Arc using the Arc agent
func (i *Installer) runArcAgentConnect(ctx context.Context) error {
	i.logger.Info("Connecting machine to Azure Arc using azcmagent")

	// Get Arc configuration details
	arcLocation := i.config.GetArcLocation()
	arcMachineName := i.config.GetArcMachineName()
	arcResourceGroup := i.config.GetArcResourceGroup()
	subscriptionID := i.config.Azure.SubscriptionID
	tenantID := i.config.Azure.TenantID

	// Get Arc tags
	tags := i.config.GetArcTags()
	tagArgs := []string{}
	for key, value := range tags {
		tagArgs = append(tagArgs, "--tags", fmt.Sprintf("%s=%s", key, value))
	}

	// Build azcmagent connect command
	args := []string{
		"azcmagent", "connect",
		"--resource-group", arcResourceGroup,
		"--tenant-id", tenantID,
		"--location", arcLocation,
		"--subscription-id", subscriptionID,
		"--resource-name", arcMachineName,
	}

	// Add tags if any
	args = append(args, tagArgs...)

	// Add authentication parameters based on available credentials
	if err := i.addAuthenticationArgs(ctx, &args); err != nil {
		return fmt.Errorf("failed to configure authentication for Arc agent: %w", err)
	}

	// Execute the command
	cmd := exec.CommandContext(ctx, "sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to connect to Azure Arc: %w, output: %s", err, string(output))
	}

	i.logger.Infof("Arc agent connect completed: %s", string(output))
	return nil
}

// AssignRBACRoles assigns required RBAC roles to the Arc machine's managed identity
func (i *Installer) assignRBACRoles(ctx context.Context, arcMachine *armhybridcompute.Machine) error {
	managedIdentityID := getArcMachineIdentityID(arcMachine)
	if managedIdentityID == "" {
		return fmt.Errorf("managed identity ID not found on Arc machine")
	}

	i.logger.Infof("Assigning roles to managed identity: %s", managedIdentityID)

	// Create role assignments client
	client, err := i.CreateRoleAssignmentsClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create role assignments client: %w", err)
	}

	// Assign each required role
	requiredRoles := i.getRoleAssignments(arcMachine)
	for _, role := range requiredRoles {
		i.logger.Infof("Assigning role '%s' to managed identity %s on scope %s", role.RoleName, managedIdentityID, role.Scope)
		if err := i.assignRole(ctx, client, managedIdentityID, role.RoleID, role.Scope, role.RoleName); err != nil {
			i.logger.Errorf("Failed to assign role '%s' on scope %s: %v", role.RoleName, role.Scope, err)
			return err
		}
	}

	i.logger.Info("All RBAC roles assigned successfully")
	return nil
}

// assignRole creates a role assignment for the given principal, role, and scope
func (i *Installer) assignRole(ctx context.Context, client *armauthorization.RoleAssignmentsClient, principalID, roleDefinitionID, scope, roleName string) error {
	// Build the full role definition ID
	fullRoleDefinitionID := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s",
		i.config.Azure.SubscriptionID, roleDefinitionID)

	// Check if assignment already exists
	hasRole, err := i.checkRoleAssignment(ctx, client, principalID, roleDefinitionID, scope)
	if err != nil {
		i.logger.Warnf("Error checking existing role assignment for %s: %v", roleName, err)
	} else if hasRole {
		i.logger.Infof("Role assignment already exists for role '%s' on scope %s", roleName, scope)
		return nil
	}

	// Generate a unique name for the role assignment (UUID format required)
	roleAssignmentName := uuid.New().String()

	// Create the role assignment
	assignment := armauthorization.RoleAssignmentCreateParameters{
		Properties: &armauthorization.RoleAssignmentProperties{
			PrincipalID:      &principalID,
			RoleDefinitionID: &fullRoleDefinitionID,
		},
	}

	_, err = client.Create(ctx, scope, roleAssignmentName, assignment, nil)
	if err != nil {
		// Check if it's a conflict error (assignment already exists)
		if strings.Contains(err.Error(), "RoleAssignmentExists") {
			i.logger.Infof("Role assignment already exists for role '%s' on scope %s", roleName, scope)
			return nil
		}
		return fmt.Errorf("failed to create role assignment: %w", err)
	}

	return nil
}

// WaitForRBACPermissions waits for RBAC permissions to be available
func (i *Installer) waitForRBACPermissions(ctx context.Context, arcMachine *armhybridcompute.Machine) error {
	i.logger.Info("Waiting for RBAC permissions to be assigned to Arc managed identity...")

	// Get Arc machine info to get the managed identity object ID
	managedIdentityID := getArcMachineIdentityID(arcMachine)
	if managedIdentityID == "" {
		return fmt.Errorf("managed identity ID not found on Arc machine")
	}

	i.logger.Infof("Checking permissions for managed identity: %s", managedIdentityID)
	i.logger.Info("Please ensure the following permissions are assigned manually:")
	i.logger.Info("  1. Reader role on the Arc machine (for Arc authentication)")
	i.logger.Info("  2. Reader role on the AKS cluster")
	i.logger.Info("  3. Azure Kubernetes Service RBAC Cluster Admin role on the AKS cluster")
	i.logger.Info("  4. Azure Kubernetes Service Cluster Admin Role on the AKS cluster")
	i.logger.Info("  5. Network Contributor role on the cluster resource group")
	i.logger.Info("  6. Contributor role on the managed cluster resource group")

	// Check permissions immediately first
	if hasPermissions := i.checkPermissionsWithLogging(ctx, managedIdentityID, true); hasPermissions {
		i.logger.Info("✅ All required RBAC permissions are already available!")
		return nil
	}

	// Start polling for permissions
	return i.pollForPermissions(ctx, managedIdentityID)
}

// checkPermissionsWithLogging checks permissions and logs the result appropriately
func (i *Installer) checkPermissionsWithLogging(ctx context.Context, managedIdentityID string, isFirstCheck bool) bool {
	i.logger.Info("Checking if required permissions are available...")

	hasPermissions, err := i.checkRequiredPermissions(ctx, managedIdentityID)
	if err != nil {
		if isFirstCheck {
			i.logger.Warnf("Error checking permissions on first attempt: %v", err)
		} else {
			i.logger.Warnf("Error checking permissions (will retry): %v", err)
		}
		return false
	}

	return hasPermissions
}

// pollForPermissions polls for RBAC permissions with timeout and interval
func (i *Installer) pollForPermissions(ctx context.Context, managedIdentityID string) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	maxWaitTime := 30 * time.Minute // Maximum wait time
	timeout := time.After(maxWaitTime)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for permissions: %w", ctx.Err())
		case <-timeout:
			return fmt.Errorf("timeout after %v waiting for RBAC permissions to be assigned", maxWaitTime)
		case <-ticker.C:
			if hasPermissions := i.checkPermissionsWithLogging(ctx, managedIdentityID, false); hasPermissions {
				i.logger.Info("✅ All required RBAC permissions are now available!")
				return nil
			}
			i.logger.Info("⏳ Some permissions are still missing, will check again in 30 seconds...")
		}
	}
}

// installPrerequisites installs required packages for Arc agent
func (i *Installer) installPrerequisites() error {
	packages := []string{"curl", "wget", "gnupg", "lsb-release", "jq", "net-tools"}

	// apt-get for Ubuntu/Debian
	if err := utils.RunSystemCommand("apt-get", "update"); err == nil {
		for _, pkg := range packages {
			if err := utils.RunSystemCommand("apt-get", "install", "-y", pkg); err != nil {
				i.logger.Warnf("Failed to install %s via apt-get: %v", pkg, err)
			}
		}
		return nil
	}

	return fmt.Errorf("unable to install prerequisites - no supported package manager found")
}

// isArcPackageCorrupted checks if the Arc agent package is corrupted (installed but files missing)
func (i *Installer) isArcPackageCorrupted() bool {
	// Check if package is installed according to dpkg
	cmd := exec.Command("dpkg", "-l", "azcmagent")
	if err := cmd.Run(); err != nil {
		// Package not installed according to dpkg
		return false
	}

	i.logger.Debug("Arc agent package is installed according to dpkg, checking file integrity")

	// Package is installed, but check if files actually exist
	for _, path := range arcPaths {
		if _, err := os.Stat(path); err == nil {
			i.logger.Debugf("Found Arc agent binary at %s", path)
			return false // Files exist, not corrupted
		}
	}

	i.logger.Warn("Arc agent package is installed but no binary files found - package corruption detected")
	return true // Package installed but files missing = corruption
}

// forceReinstallArcAgent removes and reinstalls the corrupted Arc agent package
func (i *Installer) forceReinstallArcAgent(ctx context.Context) error {
	i.logger.Info("Forcing Arc agent package reinstallation due to corruption")

	// Step 1: Remove the corrupted package
	i.logger.Info("Removing corrupted Arc agent package...")
	if err := utils.RunSystemCommand("dpkg", "--remove", "--force-remove-reinstreq", "azcmagent"); err != nil {
		i.logger.Warnf("Failed to remove package via dpkg: %v", err)
		// Try apt-get remove as fallback
		if err := utils.RunSystemCommand("apt-get", "remove", "-y", "--purge", "azcmagent"); err != nil {
			i.logger.Warnf("Failed to remove package via apt-get: %v", err)
		}
	}

	// Step 3: Download and install fresh package
	i.logger.Info("Downloading and installing fresh Arc agent...")
	if err := i.downloadArcAgentScript(ctx); err != nil {
		return fmt.Errorf("failed to download Arc agent script for reinstall: %w", err)
	}

	// Clean up script after installation
	defer i.cleanupInstallationScript()

	// Run installation script
	if err := i.runArcAgentInstallation(ctx); err != nil {
		return fmt.Errorf("failed to run Arc agent installation for reinstall: %w", err)
	}

	i.logger.Info("Arc agent package successfully reinstalled")
	return nil
}

// addAuthenticationArgs adds appropriate authentication parameters to the azcmagent command
func (i *Installer) addAuthenticationArgs(ctx context.Context, args *[]string) error {
	// Try to get credentials using the same method as other Azure SDK calls
	cred, err := i.authProvider.UserCredential(ctx, i.config)
	if err != nil {
		return fmt.Errorf("failed to get Azure credentials: %w", err)
	}

	// Get access token for Azure Resource Manager
	tokenRequestOptions := policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	}

	accessToken, err := cred.GetToken(ctx, tokenRequestOptions)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	i.logger.Info("Using access token authentication for Arc agent")
	*args = append(*args, "--access-token", accessToken.Token)
	return nil
}
