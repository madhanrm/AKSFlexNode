# AKS Flex Node Uninstall Script for Windows
# This script removes all components installed by the AKS Flex Node installation script
# Must be run as Administrator

#Requires -RunAsAdministrator

param(
    [switch]$Force,
    [switch]$KeepArcAgent,
    [switch]$KeepAzureCLI,
    [switch]$KeepConfig
)

$ErrorActionPreference = "Stop"

# Configuration (should match install.ps1)
$ServiceName = "aks-flex-node"
$TaskName = "AKS Flex Node Agent"
$InstallDir = "C:\Program Files\aks-flex-node"
$ConfigDir = "C:\ProgramData\aks-flex-node\config"
$DataDir = "C:\ProgramData\aks-flex-node\data"
$LogDir = "C:\ProgramData\aks-flex-node\logs"
$BaseDir = "C:\ProgramData\aks-flex-node"

# Kubernetes directories created during bootstrap
$KubernetesDirs = @(
    "C:\k",
    "C:\etc\kubernetes",
    "C:\var\lib\kubelet",
    "C:\var\lib\cni",
    "C:\var\log\kubelet",
    "C:\var\log\calico",
    "C:\CalicoWindows",
    "C:\etc\CalicoWindows"
)

# Colors for output
function Write-Info($Message) {
    Write-Host "INFO: " -ForegroundColor Blue -NoNewline
    Write-Host $Message
}

function Write-Success($Message) {
    Write-Host "SUCCESS: " -ForegroundColor Green -NoNewline
    Write-Host $Message
}

function Write-Warning($Message) {
    Write-Host "WARNING: " -ForegroundColor Yellow -NoNewline
    Write-Host $Message
}

function Write-Error($Message) {
    Write-Host "ERROR: " -ForegroundColor Red -NoNewline
    Write-Host $Message
}

function Confirm-Uninstall {
    Write-Host "AKS Flex Node Uninstaller for Windows" -ForegroundColor Yellow
    Write-Host "======================================" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "This will remove the following components:"
    Write-Host "  * AKS Flex Node binary ($InstallDir\aks-flex-node.exe)"
    Write-Host "  * Scheduled task ($TaskName)"
    Write-Host "  * Configuration directory ($ConfigDir)"
    Write-Host "  * Data directory ($DataDir)"
    Write-Host "  * Log directory ($LogDir)"
    Write-Host "  * Firewall rules (AKS Flex Node - *)"
    Write-Host "  * Kubernetes directories (C:\k, C:\etc\kubernetes, etc.)"
    if (-not $KeepArcAgent) {
        Write-Host "  * Azure Arc agent and connection"
    }
    Write-Host ""
    Write-Host "NOTE: This will first run 'aks-flex-node unbootstrap' to clean up cluster and Arc resources." -ForegroundColor Yellow
    Write-Host ""
    
    if (-not $Force) {
        $response = Read-Host "Are you sure you want to continue? (y/N)"
        if ($response -notmatch "^[Yy]$") {
            Write-Host "Uninstall cancelled."
            exit 0
        }
    } else {
        Write-Info "Force flag provided, skipping confirmation."
    }
}

function Stop-Services {
    Write-Info "Stopping services and scheduled tasks..."
    
    # Stop scheduled task
    $task = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
    if ($task) {
        if ($task.State -eq 'Running') {
            Write-Info "Stopping scheduled task: $TaskName"
            Stop-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
        }
    }
    
    # Stop Windows service if it exists
    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($service) {
        if ($service.Status -eq 'Running') {
            Write-Info "Stopping service: $ServiceName"
            Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
        }
    }
    
    # Stop kubelet service
    $kubeletService = Get-Service -Name "kubelet" -ErrorAction SilentlyContinue
    if ($kubeletService) {
        if ($kubeletService.Status -eq 'Running') {
            Write-Info "Stopping kubelet service..."
            Stop-Service -Name "kubelet" -Force -ErrorAction SilentlyContinue
        }
    }
    
    # Stop containerd service
    $containerdService = Get-Service -Name "containerd" -ErrorAction SilentlyContinue
    if ($containerdService) {
        if ($containerdService.Status -eq 'Running') {
            Write-Info "Stopping containerd service..."
            Stop-Service -Name "containerd" -Force -ErrorAction SilentlyContinue
        }
    }
    
    # Stop Calico services
    $calicoServices = @("CalicoFelix", "CalicoNode", "calico-node")
    foreach ($svcName in $calicoServices) {
        $svc = Get-Service -Name $svcName -ErrorAction SilentlyContinue
        if ($svc -and $svc.Status -eq 'Running') {
            Write-Info "Stopping $svcName service..."
            Stop-Service -Name $svcName -Force -ErrorAction SilentlyContinue
        }
    }
    
    Write-Success "Services stopped"
}

function Invoke-Unbootstrap {
    Write-Info "Running unbootstrap to clean up cluster and Arc resources..."
    
    $binaryPath = Join-Path $InstallDir "aks-flex-node.exe"
    $configPath = Join-Path $ConfigDir "config.json"
    
    # Check if binary exists
    if (-not (Test-Path $binaryPath)) {
        Write-Warning "AKS Flex Node binary not found at $binaryPath"
        Write-Info "Skipping unbootstrap - binary may already be removed"
        return
    }
    
    # Check if config exists
    if (-not (Test-Path $configPath)) {
        Write-Warning "Config file not found at $configPath"
        Write-Warning "Cannot run unbootstrap without config file - skipping resource cleanup"
        Write-Info "Manual cleanup of Azure resources may be required"
        return
    }
    
    Write-Info "Using config file: $configPath"
    
    try {
        $process = Start-Process -FilePath $binaryPath -ArgumentList "unbootstrap", "--config", "`"$configPath`"" -Wait -NoNewWindow -PassThru
        if ($process.ExitCode -ne 0) {
            Write-Warning "Unbootstrap exited with code $($process.ExitCode) - this may be expected if resources are already cleaned up"
        } else {
            Write-Success "Unbootstrap completed successfully"
        }
    } catch {
        Write-Warning "Unbootstrap failed: $_ - this may be expected if resources are already cleaned up"
    }
}

function Remove-ScheduledTask {
    Write-Info "Removing scheduled task..."
    
    $task = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
    if ($task) {
        Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
        Write-Success "Removed scheduled task: $TaskName"
    } else {
        Write-Info "Scheduled task not found: $TaskName"
    }
}

function Remove-WindowsService {
    Write-Info "Removing Windows services..."
    
    # Remove AKS Flex Node service
    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($service) {
        if ($service.Status -eq 'Running') {
            Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
        }
        sc.exe delete $ServiceName | Out-Null
        Write-Success "Removed service: $ServiceName"
    }
    
    # Remove kubelet service
    $kubeletService = Get-Service -Name "kubelet" -ErrorAction SilentlyContinue
    if ($kubeletService) {
        if ($kubeletService.Status -eq 'Running') {
            Stop-Service -Name "kubelet" -Force -ErrorAction SilentlyContinue
        }
        sc.exe delete "kubelet" | Out-Null
        Write-Success "Removed kubelet service"
    }
    
    # Remove containerd service
    $containerdService = Get-Service -Name "containerd" -ErrorAction SilentlyContinue
    if ($containerdService) {
        if ($containerdService.Status -eq 'Running') {
            Stop-Service -Name "containerd" -Force -ErrorAction SilentlyContinue
        }
        sc.exe delete "containerd" | Out-Null
        Write-Success "Removed containerd service"
    }
    
    # Remove Calico services
    $calicoServices = @("CalicoFelix", "CalicoNode", "calico-node")
    foreach ($svcName in $calicoServices) {
        $svc = Get-Service -Name $svcName -ErrorAction SilentlyContinue
        if ($svc) {
            if ($svc.Status -eq 'Running') {
                Stop-Service -Name $svcName -Force -ErrorAction SilentlyContinue
            }
            sc.exe delete $svcName | Out-Null
            Write-Success "Removed $svcName service"
        }
    }
}

function Remove-FirewallRules {
    Write-Info "Removing firewall rules..."
    
    $rules = Get-NetFirewallRule -DisplayName "AKS Flex Node*" -ErrorAction SilentlyContinue
    if ($rules) {
        foreach ($rule in $rules) {
            Remove-NetFirewallRule -DisplayName $rule.DisplayName -ErrorAction SilentlyContinue
            Write-Info "Removed firewall rule: $($rule.DisplayName)"
        }
        Write-Success "Firewall rules removed"
    } else {
        Write-Info "No AKS Flex Node firewall rules found"
    }
    
    # Remove Calico-related firewall rules
    $calicoRules = Get-NetFirewallRule -DisplayName "*Calico*" -ErrorAction SilentlyContinue
    if ($calicoRules) {
        foreach ($rule in $calicoRules) {
            Remove-NetFirewallRule -DisplayName $rule.DisplayName -ErrorAction SilentlyContinue
            Write-Info "Removed firewall rule: $($rule.DisplayName)"
        }
    }
}

function Remove-HNSNetworks {
    Write-Info "Removing HNS networks..."
    
    try {
        # Import HNS module if available
        $hnsModule = Get-Module -Name HNS -ListAvailable -ErrorAction SilentlyContinue
        if (-not $hnsModule) {
            # Try to load from containerd directory
            $hnsPath = "C:\Program Files\containerd\hns.psm1"
            if (Test-Path $hnsPath) {
                Import-Module $hnsPath -ErrorAction SilentlyContinue
            }
        }
        
        # Get and remove Calico networks
        $networks = Get-HnsNetwork -ErrorAction SilentlyContinue | Where-Object { $_.Name -like "*Calico*" -or $_.Name -like "*External*" }
        foreach ($network in $networks) {
            Write-Info "Removing HNS network: $($network.Name)"
            Remove-HnsNetwork -Id $network.Id -ErrorAction SilentlyContinue
        }
        
        Write-Success "HNS networks removed"
    } catch {
        Write-Warning "Failed to remove HNS networks: $_ (may require manual cleanup)"
    }
}

function Remove-Directories {
    Write-Info "Removing directories..."
    
    # Remove AKS Flex Node directories
    $directories = @($ConfigDir, $DataDir, $LogDir, $BaseDir)
    
    if (-not $KeepConfig) {
        foreach ($dir in $directories) {
            if (Test-Path $dir) {
                Write-Info "Removing directory: $dir"
                Remove-Item -Path $dir -Recurse -Force -ErrorAction SilentlyContinue
                Write-Success "Removed: $dir"
            }
        }
    } else {
        Write-Info "Keeping configuration directory (--KeepConfig specified)"
    }
    
    # Remove Kubernetes directories
    foreach ($dir in $KubernetesDirs) {
        if (Test-Path $dir) {
            Write-Info "Removing Kubernetes directory: $dir"
            Remove-Item -Path $dir -Recurse -Force -ErrorAction SilentlyContinue
            Write-Success "Removed: $dir"
        }
    }
    
    # Remove containerd directories
    $containerdDirs = @("C:\Program Files\containerd", "C:\ProgramData\containerd")
    foreach ($containerdDir in $containerdDirs) {
        if (Test-Path $containerdDir) {
            Write-Info "Removing containerd directory: $containerdDir"
            Remove-Item -Path $containerdDir -Recurse -Force -ErrorAction SilentlyContinue
            Write-Success "Removed: $containerdDir"
        }
    }
}

function Remove-Binary {
    Write-Info "Removing binary..."
    
    $binaryPath = Join-Path $InstallDir "aks-flex-node.exe"
    
    if (Test-Path $binaryPath) {
        Remove-Item -Path $binaryPath -Force -ErrorAction SilentlyContinue
        Write-Success "Removed binary: $binaryPath"
    } else {
        Write-Info "Binary not found: $binaryPath"
    }
    
    # Remove installation directory if empty
    if (Test-Path $InstallDir) {
        $items = Get-ChildItem -Path $InstallDir -ErrorAction SilentlyContinue
        if (-not $items) {
            Remove-Item -Path $InstallDir -Force -ErrorAction SilentlyContinue
            Write-Info "Removed empty installation directory: $InstallDir"
        }
    }
    
    # Remove from PATH
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    if ($currentPath -like "*$InstallDir*") {
        Write-Info "Removing $InstallDir from system PATH..."
        $newPath = ($currentPath.Split(';') | Where-Object { $_ -ne $InstallDir -and $_ -ne "" }) -join ';'
        [Environment]::SetEnvironmentVariable("Path", $newPath, "Machine")
        Write-Success "Removed from PATH"
    }
}

function Remove-ArcAgent {
    if ($KeepArcAgent) {
        Write-Info "Keeping Azure Arc agent (--KeepArcAgent specified)"
        return
    }
    
    Write-Info "Removing Azure Arc agent..."
    
    $azcmagentPath = "C:\Program Files\AzureConnectedMachineAgent\azcmagent.exe"
    
    if (Test-Path $azcmagentPath) {
        # Disconnect from Arc (force local only in case Azure connection fails)
        Write-Info "Disconnecting from Azure Arc..."
        try {
            & $azcmagentPath disconnect --force-local-only 2>$null
        } catch {
            Write-Warning "Arc disconnect failed (may already be disconnected)"
        }
        
        # Stop Arc services
        Write-Info "Stopping Arc agent services..."
        $arcServices = @("himds", "GCArcService", "ExtensionService")
        foreach ($svcName in $arcServices) {
            $svc = Get-Service -Name $svcName -ErrorAction SilentlyContinue
            if ($svc) {
                Stop-Service -Name $svcName -Force -ErrorAction SilentlyContinue
            }
        }
        
        # Uninstall Arc agent via MSI
        Write-Info "Uninstalling Azure Arc agent..."
        $arcProduct = Get-WmiObject -Class Win32_Product | Where-Object { $_.Name -like "*Azure Connected Machine Agent*" }
        if ($arcProduct) {
            $arcProduct.Uninstall() | Out-Null
            Write-Success "Azure Arc agent uninstalled"
        } else {
            # Try removing via msiexec with product code
            $uninstallKey = Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*" -ErrorAction SilentlyContinue | 
                            Where-Object { $_.DisplayName -like "*Azure Connected Machine Agent*" }
            if ($uninstallKey) {
                $productCode = $uninstallKey.PSChildName
                Start-Process msiexec.exe -ArgumentList "/x $productCode /quiet /norestart" -Wait -NoNewWindow
                Write-Success "Azure Arc agent uninstalled"
            } else {
                Write-Warning "Could not find Arc agent uninstaller - manual removal may be required"
            }
        }
        
        # Clean up Arc directories
        $arcDirs = @(
            "C:\Program Files\AzureConnectedMachineAgent",
            "C:\ProgramData\AzureConnectedMachineAgent",
            "$env:ProgramData\GuestConfig"
        )
        foreach ($dir in $arcDirs) {
            if (Test-Path $dir) {
                Remove-Item -Path $dir -Recurse -Force -ErrorAction SilentlyContinue
                Write-Info "Removed: $dir"
            }
        }
        
        Write-Success "Azure Arc agent removed"
    } else {
        Write-Info "Azure Arc agent not found - already removed or never installed"
    }
}

function Show-CompletionMessage {
    Write-Host ""
    Write-Success "AKS Flex Node uninstallation completed!"
    Write-Host ""
    Write-Host "What was removed:" -ForegroundColor Yellow
    Write-Host "  [OK] AKS Flex Node binary" -ForegroundColor Green
    Write-Host "  [OK] Scheduled task and Windows services" -ForegroundColor Green
    Write-Host "  [OK] Firewall rules" -ForegroundColor Green
    Write-Host "  [OK] HNS networks" -ForegroundColor Green
    Write-Host "  [OK] Kubernetes directories" -ForegroundColor Green
    if (-not $KeepConfig) {
        Write-Host "  [OK] Configuration and data directories" -ForegroundColor Green
    }
    if (-not $KeepArcAgent) {
        Write-Host "  [OK] Azure Arc agent and connection" -ForegroundColor Green
    }
    Write-Host ""
    Write-Host "The system has been returned to its pre-installation state." -ForegroundColor Green
    Write-Host ""
    Write-Host "Note: A system restart may be required to complete cleanup of some components." -ForegroundColor Yellow
}

# Main function
function Main {
    Write-Host "AKS Flex Node Uninstaller for Windows" -ForegroundColor Green
    Write-Host "======================================" -ForegroundColor Green
    Write-Host ""
    
    # Confirm uninstall
    Confirm-Uninstall
    
    Write-Host ""
    Write-Info "Starting AKS Flex Node uninstallation..."
    
    # Run unbootstrap first
    Invoke-Unbootstrap
    
    # Stop all services
    Stop-Services
    
    # Remove in reverse order of installation
    Remove-ScheduledTask
    Remove-WindowsService
    Remove-FirewallRules
    Remove-HNSNetworks
    Remove-ArcAgent
    Remove-Directories
    Remove-Binary
    
    # Show completion message
    Show-CompletionMessage
}

# Run main
Main
