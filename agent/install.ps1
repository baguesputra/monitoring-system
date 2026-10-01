<#>
.SYNOPSIS
    Installs Seneschal Monitoring Agent as a hardened Windows Service.

.DESCRIPTION
    This script installs the Seneschal agent to C:\ProgramData\Seneschal\,
    hardens NTFS ACLs (SYSTEM/Admins full, Users read-only), registers
    the agent as a Windows Service running as SYSTEM with auto-restart,
    and starts the service.

.PARAMETER CollectorURL
    Collector API base URL (e.g., http://192.168.1.50:8081)

.PARAMETER ServerID
    Unique device identifier (defaults to BIOS Serial Number if omitted)

.PARAMETER DeviceType
    Device type: server, pc, or laptop (default: pc)

.PARAMETER IntervalSeconds
    Telemetry interval in seconds (default: 15)

.PARAMETER ServicesToCheck
    Comma-separated list of process names to monitor (default: "nginx,postgres")

.PARAMETER SkipAcl
    Skip NTFS ACL hardening (for testing only)

.EXAMPLE
    .\install.ps1 -CollectorURL "http://192.168.1.50:8081" -ServerID "PC-HRD-01" -DeviceType "pc"

.NOTES
    Requires Administrator privileges. Run PowerShell as Administrator.
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)]
    [ValidatePattern('^https?://')]
    [string]$CollectorURL,

    [Parameter()]
    [string]$ServerID = "",

    [Parameter()]
    [ValidateSet('server','pc','laptop')]
    [string]$DeviceType = "pc",

    [Parameter()]
    [int]$IntervalSeconds = 15,

    [Parameter()]
    [string]$ServicesToCheck = "nginx,postgres",

    [Parameter()]
    [switch]$SkipAcl
)

# Check for Admin rights
if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Error "This script requires Administrator privileges. Please run PowerShell as Administrator."
    exit 1
}

$ErrorActionPreference = "Stop"

$InstallDir = "C:\ProgramData\Seneschal"
$BinaryName = "agent.exe"
$ConfigName = "config.yaml"
$SourceDir = $PSScriptRoot

Write-Host "=== Seneschal Agent Installation ===" -ForegroundColor Cyan
Write-Host "Install directory: $InstallDir" -ForegroundColor Gray

# 1. Create install directory
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Write-Host "Created directory: $InstallDir" -ForegroundColor Green
}

# 2. Copy agent binary
$SourceBinary = Join-Path $SourceDir $BinaryName
$DestBinary = Join-Path $InstallDir $BinaryName

if (-not (Test-Path $SourceBinary)) {
    Write-Error "Source binary not found: $SourceBinary"
    exit 1
}

Copy-Item -Path $SourceBinary -Destination $DestBinary -Force
Write-Host "Copied $BinaryName to $InstallDir" -ForegroundColor Green

# 3. Generate config.yaml
$ConfigPath = Join-Path $InstallDir $ConfigName
if (-not $ServerID) {
    $bios = Get-WmiObject Win32_BIOS -ErrorAction SilentlyContinue
    $ServerID = $bios.SerialNumber.Trim()
    if (-not $ServerID) { $ServerID = $env:COMPUTERNAME }
    Write-Host "Auto-generated ServerID from BIOS: $ServerID" -ForegroundColor Yellow
}

$ConfigContent = @"
server_id: "$ServerID"
device_type: "$DeviceType"
collector_url: "$CollectorURL"
interval_seconds: $IntervalSeconds
services_to_check:
$(($ServicesToCheck -split ',').ForEach({ "  - `"$_.Trim()`"" }) -join "`n")
"@

Set-Content -Path $ConfigPath -Value $ConfigContent -Encoding UTF8
Write-Host "Generated $ConfigName" -ForegroundColor Green

# 4. Harden NTFS ACLs (unless -SkipAcl)
if (-not $SkipAcl) {
    Write-Host "Hardening NTFS ACLs..." -ForegroundColor Cyan
    
    $Acl = Get-Acl -Path $InstallDir
    
    # Disable inheritance
    $Acl.SetAccessRuleProtection($true, $false)
    
    # Remove existing rules
    $Acl.Access | ForEach-Object { $Acl.RemoveAccessRule($_) }
    
    # SYSTEM - Full Control
    $SystemRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
        "SYSTEM", "FullControl", "ContainerInherit,ObjectInherit", "None", "Allow"
    )
    $Acl.AddAccessRule($SystemRule)
    
    # Administrators - Full Control
    $AdminRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
        "Administrators", "FullControl", "ContainerInherit,ObjectInherit", "None", "Allow"
    )
    $Acl.AddAccessRule($AdminRule)
    
    # Users - Read & Execute only (no write/modify)
    $UsersRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
        "Users", "ReadAndExecute", "ContainerInherit,ObjectInherit", "None", "Allow"
    )
    $Acl.AddAccessRule($UsersRule)
    
    Set-Acl -Path $InstallDir -AclObject $Acl
    Write-Host "NTFS ACLs hardened: SYSTEM/Admins=Full, Users=ReadOnly" -ForegroundColor Green
}

# 5. Register Windows Service
$ServiceName = "SeneschalAgent"
$DisplayName = "Seneschal Monitoring Agent"
$Description = "Seneschal Infrastructure Monitoring Agent - collects telemetry & asset inventory"

# Remove existing service if present
$Existing = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($Existing) {
    Write-Host "Removing existing service..." -ForegroundColor Yellow
    sc.exe stop $ServiceName 2>$null
    sc.exe delete $ServiceName 2>$null
    Start-Sleep -Seconds 2
}

Write-Host "Registering Windows Service..." -ForegroundColor Cyan

# Use sc.exe for service creation (more reliable than New-Service)
$ServicePath = "`"$DestBinary`""
sc.exe create $ServiceName `
    binPath= $ServicePath `
    DisplayName= $DisplayName `
    start= auto `
    obj= "LocalSystem" `
    type= own `
    error= normal

# Set description
sc.exe description $ServiceName $Description

# Configure recovery actions (restart on failure)
sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/10000/restart/60000

Write-Host "Service registered: $ServiceName" -ForegroundColor Green

# 6. Start the service
Write-Host "Starting service..." -ForegroundColor Cyan
sc.exe start $ServiceName

Start-Sleep -Seconds 3

$Status = (Get-Service -Name $ServiceName).Status
if ($Status -eq 'Running') {
    Write-Host "[OK] Service is RUNNING" -ForegroundColor Green
} else {
    Write-Warning 'Service status: {0} (check Event Viewer for details)' -f $Status -ForegroundColor Yellow
}

Write-Host ''
Write-Host '=== Installation Complete ===' -ForegroundColor Cyan
Write-Host 'Install dir: ' + $InstallDir -ForegroundColor Gray
Write-Host 'Config:      ' + $ConfigPath -ForegroundColor Gray
Write-Host 'Service:     ' + $ServiceName + ' (Auto-start, SYSTEM)' -ForegroundColor Gray
Write-Host 'To uninstall: Run .\uninstall.ps1 as Administrator' -ForegroundColor Gray