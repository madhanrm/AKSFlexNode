# AKS Flex Node Installation Script for Windows
# This script downloads and installs the latest AKS Flex Node binary from GitHub releases
# Must be run as Administrator

#Requires -RunAsAdministrator

param(
    [string]$Version = "",
    [switch]$Force,
    [switch]$SkipArcAgent,
    [switch]$SkipAzureCLI
)

$ErrorActionPreference = "Stop"

# Configuration
$Repo = "Azure/AKSFlexNode"
$ServiceName = "aks-flex-node"
$InstallDir = "C:\Program Files\aks-flex-node"
$ConfigDir = "C:\ProgramData\aks-flex-node\config"
$DataDir = "C:\ProgramData\aks-flex-node\data"
$LogDir = "C:\ProgramData\aks-flex-node\logs"
$GitHubAPI = "https://api.github.com/repos/$Repo"
$GitHubReleases = "$GitHubAPI/releases"

# Colors for output (using Write-Host colors)
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

function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default {
            Write-Error "Unsupported architecture: $arch"
            Write-Error "AKS Flex Node supports: AMD64 (amd64), ARM64 (arm64)"
            exit 1
        }
    }
}

function Test-WindowsVersion {
    $osInfo = Get-CimInstance Win32_OperatingSystem
    $version = [System.Environment]::OSVersion.Version
    
    Write-Info "Detected: $($osInfo.Caption) (Build $($version.Build))"
    
    # Check for Windows Server 2019, 2022, or Windows 10/11
    if ($version.Build -ge 17763) {
        Write-Info "Windows version is supported"
        return $true
    } else {
        Write-Warning "Windows version may not be fully supported"
        Write-Warning "AKS Flex Node is tested on Windows Server 2019, 2022, and Windows 10/11"
        return $true  # Continue anyway
    }
}

function Get-LatestRelease {
    Write-Info "Fetching latest release information..."
    
    $latestReleaseUrl = "$GitHubReleases/latest"
    
    try {
        $response = Invoke-RestMethod -Uri $latestReleaseUrl -UseBasicParsing
        return $response.tag_name
    } catch {
        Write-Error "Failed to fetch latest release: $_"
        return $null
    }
}

function Get-SpecificRelease($Tag) {
    Write-Info "Fetching release information for $Tag..."
    
    $releaseUrl = "$GitHubReleases/tags/$Tag"
    
    try {
        $response = Invoke-RestMethod -Uri $releaseUrl -UseBasicParsing
        return $response.tag_name
    } catch {
        Write-Error "Failed to fetch release $Tag : $_"
        return $null
    }
}

function Download-Binary($Version, $Arch) {
    $binaryName = "aks-flex-node-windows-$Arch"
    $archiveName = "$binaryName.zip"
    $downloadUrl = "https://github.com/$Repo/releases/download/$Version/$archiveName"
    
    Write-Info "Downloading AKS Flex Node $Version for windows/$Arch..."
    Write-Info "Download URL: $downloadUrl"
    
    $tempDir = Join-Path $env:TEMP "aks-flex-node-install"
    if (Test-Path $tempDir) {
        Remove-Item -Path $tempDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    
    $archivePath = Join-Path $tempDir $archiveName
    
    try {
        # Use TLS 1.2
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        
        Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing
        
        Write-Info "Extracting binary..."
        Expand-Archive -Path $archivePath -DestinationPath $tempDir -Force
        
        $binaryPath = Join-Path $tempDir "$binaryName.exe"
        if (-not (Test-Path $binaryPath)) {
            # Try without extension in archive
            $binaryPath = Join-Path $tempDir $binaryName
            if (Test-Path $binaryPath) {
                Rename-Item -Path $binaryPath -NewName "$binaryName.exe"
                $binaryPath = Join-Path $tempDir "$binaryName.exe"
            }
        }
        
        if (-not (Test-Path $binaryPath)) {
            Write-Error "Binary not found in archive"
            Remove-Item -Path $tempDir -Recurse -Force
            exit 1
        }
        
        return $binaryPath
    } catch {
        Write-Error "Failed to download binary: $_"
        if (Test-Path $tempDir) {
            Remove-Item -Path $tempDir -Recurse -Force
        }
        exit 1
    }
}

function Install-Binary($BinaryPath) {
    Write-Info "Installing binary to $InstallDir..."
    
    # Create installation directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    
    # Stop existing service if running
    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($service -and $service.Status -eq 'Running') {
        Write-Info "Stopping existing service..."
        Stop-Service -Name $ServiceName -Force
    }
    
    # Copy binary
    $destPath = Join-Path $InstallDir "aks-flex-node.exe"
    Copy-Item -Path $BinaryPath -Destination $destPath -Force
    
    # Add to PATH if not already there
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    if ($currentPath -notlike "*$InstallDir*") {
        Write-Info "Adding $InstallDir to system PATH..."
        [Environment]::SetEnvironmentVariable("Path", "$currentPath;$InstallDir", "Machine")
    }
    
    Write-Success "Binary installed to $destPath"
}

function Setup-Directories {
    Write-Info "Creating directories..."
    
    $directories = @($ConfigDir, $DataDir, $LogDir)
    
    foreach ($dir in $directories) {
        if (-not (Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
            Write-Info "Created: $dir"
        }
    }
    
    Write-Success "Directories created successfully"
}

function Install-AzureCLI {
    if ($SkipAzureCLI) {
        Write-Info "Skipping Azure CLI installation (--SkipAzureCLI specified)"
        return
    }
    
    Write-Info "Checking Azure CLI installation..."
    
    $azCmd = Get-Command az -ErrorAction SilentlyContinue
    if ($azCmd) {
        Write-Info "Azure CLI already installed"
        return
    }
    
    Write-Info "Installing Azure CLI..."
    
    $msiUrl = "https://aka.ms/installazurecliwindows"
    $msiPath = Join-Path $env:TEMP "AzureCLI.msi"
    
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        Invoke-WebRequest -Uri $msiUrl -OutFile $msiPath -UseBasicParsing
        
        Write-Info "Running Azure CLI installer..."
        Start-Process msiexec.exe -ArgumentList "/i `"$msiPath`" /quiet /norestart" -Wait -NoNewWindow
        
        Remove-Item -Path $msiPath -Force -ErrorAction SilentlyContinue
        
        # Refresh PATH
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
        
        Write-Success "Azure CLI installed successfully"
    } catch {
        Write-Warning "Failed to install Azure CLI: $_"
        Write-Warning "You may need to install Azure CLI manually"
    }
}

function Install-ArcAgent {
    if ($SkipArcAgent) {
        Write-Info "Skipping Azure Arc agent installation (--SkipArcAgent specified)"
        return
    }
    
    Write-Info "Checking Azure Arc agent installation..."
    
    $azcmagentPath = "C:\Program Files\AzureConnectedMachineAgent\azcmagent.exe"
    if (Test-Path $azcmagentPath) {
        Write-Info "Azure Arc agent already installed"
        return
    }
    
    Write-Info "Installing Azure Arc agent..."
    
    $arcInstallerUrl = "https://aka.ms/AzureConnectedMachineAgent"
    $installerPath = Join-Path $env:TEMP "AzureConnectedMachineAgent.msi"
    
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        Invoke-WebRequest -Uri $arcInstallerUrl -OutFile $installerPath -UseBasicParsing
        
        Write-Info "Running Azure Arc agent installer..."
        Start-Process msiexec.exe -ArgumentList "/i `"$installerPath`" /quiet /norestart" -Wait -NoNewWindow
        
        Remove-Item -Path $installerPath -Force -ErrorAction SilentlyContinue
        
        if (Test-Path $azcmagentPath) {
            Write-Success "Azure Arc agent installed successfully"
        } else {
            Write-Warning "Azure Arc agent installation may have failed"
        }
    } catch {
        Write-Warning "Failed to install Azure Arc agent: $_"
        Write-Warning "You may need to install Azure Arc agent manually"
    }
}

function Configure-Firewall {
    Write-Info "Configuring Windows Firewall rules..."
    
    # Kubelet API port
    $ruleName = "AKS Flex Node - Kubelet API"
    $existingRule = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
    if (-not $existingRule) {
        New-NetFirewallRule -DisplayName $ruleName -Direction Inbound -Protocol TCP -LocalPort 10250 -Action Allow | Out-Null
        Write-Info "Created firewall rule: $ruleName"
    }
    
    # Calico VXLAN
    $ruleName = "AKS Flex Node - Calico VXLAN"
    $existingRule = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
    if (-not $existingRule) {
        New-NetFirewallRule -DisplayName $ruleName -Direction Inbound -Protocol UDP -LocalPort 4789 -Action Allow | Out-Null
        Write-Info "Created firewall rule: $ruleName"
    }
    
    # Calico BGP (optional)
    $ruleName = "AKS Flex Node - Calico BGP"
    $existingRule = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
    if (-not $existingRule) {
        New-NetFirewallRule -DisplayName $ruleName -Direction Inbound -Protocol TCP -LocalPort 179 -Action Allow | Out-Null
        Write-Info "Created firewall rule: $ruleName"
    }
    
    Write-Success "Firewall rules configured"
}

function Register-WindowsService($Version) {
    Write-Info "Registering Windows service..."
    
    $binaryPath = Join-Path $InstallDir "aks-flex-node.exe"
    $configPath = Join-Path $ConfigDir "config.json"
    
    # Check if service already exists
    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($service) {
        Write-Info "Service already exists, updating..."
        Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
        sc.exe delete $ServiceName | Out-Null
        Start-Sleep -Seconds 2
    }
    
    # Create service using sc.exe
    $serviceBinary = "`"$binaryPath`" agent --config `"$configPath`""
    
    # Note: For production, consider using NSSM or a proper service wrapper
    # Windows services have specific requirements that Go binaries don't natively satisfy
    Write-Warning "Windows service registration requires a service wrapper"
    Write-Info "Recommended: Use NSSM (Non-Sucking Service Manager) to run as service"
    Write-Info "  1. Download NSSM from https://nssm.cc/"
    Write-Info "  2. Run: nssm install $ServiceName `"$binaryPath`" agent --config `"$configPath`""
    Write-Info "  3. Configure service settings with: nssm edit $ServiceName"
    
    # Create a scheduled task as alternative
    Write-Info "Creating scheduled task as an alternative..."
    
    $taskName = "AKS Flex Node Agent"
    $existingTask = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
    if ($existingTask) {
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
    }
    
    $action = New-ScheduledTaskAction -Execute $binaryPath -Argument "agent --config `"$configPath`""
    $trigger = New-ScheduledTaskTrigger -AtStartup
    $principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1)
    
    Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Principal $principal -Settings $settings | Out-Null
    
    Write-Success "Scheduled task '$taskName' created"
}

function Show-NextSteps($Version) {
    Write-Success "AKS Flex Node installation completed successfully!"
    Write-Host ""
    Write-Host "Next Steps:" -ForegroundColor Yellow
    Write-Host "1. Create configuration file: $ConfigDir\config.json"
    Write-Host ""
    Write-Host "Example configuration:" -ForegroundColor Yellow
    Write-Host @"
{
  "azure": {
    "subscriptionId": "YOUR_SUBSCRIPTION_ID",
    "tenantId": "YOUR_TENANT_ID",
    "cloud": "AzurePublicCloud",
    "arc": {
      "machineName": "YOUR_MACHINE_NAME",
      "tags": {
        "node-type": "edge"
      },
      "resourceGroup": "YOUR_RESOURCE_GROUP",
      "location": "YOUR_LOCATION"
    },
    "targetCluster": {
      "resourceId": "/subscriptions/YOUR_SUBSCRIPTION_ID/resourceGroups/YOUR_RESOURCE_GROUP/providers/Microsoft.ContainerService/managedClusters/YOUR_CLUSTER_NAME",
      "location": "YOUR_LOCATION"
    }
  },
  "agent": {
    "logLevel": "info",
    "logDir": "C:\\ProgramData\\aks-flex-node\\logs"
  }
}
"@
    Write-Host ""
    Write-Host "Usage Options:" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Command Line Usage:" -ForegroundColor Blue
    Write-Host "  Run agent daemon:       aks-flex-node agent --config $ConfigDir\config.json"
    Write-Host "  Bootstrap node:         aks-flex-node bootstrap --config $ConfigDir\config.json"
    Write-Host "  Unbootstrap node:       aks-flex-node unbootstrap --config $ConfigDir\config.json"
    Write-Host "  Check version:          aks-flex-node version"
    Write-Host ""
    Write-Host "Scheduled Task:" -ForegroundColor Blue
    Write-Host "  Start agent:            Start-ScheduledTask -TaskName 'AKS Flex Node Agent'"
    Write-Host "  Stop agent:             Stop-ScheduledTask -TaskName 'AKS Flex Node Agent'"
    Write-Host "  Check status:           Get-ScheduledTask -TaskName 'AKS Flex Node Agent'"
    Write-Host ""
    Write-Host "Directories:" -ForegroundColor Yellow
    Write-Host "  Configuration: $ConfigDir"
    Write-Host "  Data:          $DataDir"
    Write-Host "  Logs:          $LogDir"
    Write-Host "  Binary:        $InstallDir\aks-flex-node.exe"
    Write-Host ""
    Write-Host "Uninstall:" -ForegroundColor Yellow
    Write-Host "  To uninstall:  Invoke-WebRequest -Uri 'https://raw.githubusercontent.com/$Repo/$Version/scripts/uninstall.ps1' | Invoke-Expression"
}

# Main function
function Main {
    Write-Host "AKS Flex Node Installer for Windows" -ForegroundColor Green
    Write-Host "====================================" -ForegroundColor Green
    Write-Host ""
    
    # Check Windows version
    Test-WindowsVersion
    
    # Detect architecture
    $arch = Get-Architecture
    Write-Info "Detected platform: windows/$arch"
    
    # Get version
    $releaseVersion = $null
    if ($Version) {
        $releaseVersion = Get-SpecificRelease -Tag $Version
    } else {
        $releaseVersion = Get-LatestRelease
    }
    
    if (-not $releaseVersion) {
        Write-Error "Failed to get release information"
        exit 1
    }
    
    Write-Info "Version: $releaseVersion"
    
    # Download binary
    $binaryPath = Download-Binary -Version $releaseVersion -Arch $arch
    
    # Install binary
    Install-Binary -BinaryPath $binaryPath
    
    # Setup directories
    Setup-Directories
    
    # Install dependencies
    Install-AzureCLI
    Install-ArcAgent
    
    # Configure firewall
    Configure-Firewall
    
    # Register service
    Register-WindowsService -Version $releaseVersion
    
    # Cleanup
    $tempDir = Split-Path $binaryPath -Parent
    if (Test-Path $tempDir) {
        Remove-Item -Path $tempDir -Recurse -Force
    }
    
    # Show next steps
    Show-NextSteps -Version $releaseVersion
}

# Run main
Main
