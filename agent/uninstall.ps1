<#>
.SYNOPSIS
    Uninstalls Seneschal Monitoring Agent and removes Windows Service.

.DESCRIPTION
    Stops and removes the SeneschalAgent Windows Service,
    and optionally removes the installation directory.
#>

[CmdletBinding()]
param(
    [Parameter()]
    [switch]$KeepData
)

# Check for Admin rights
if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Error "This script requires Administrator privileges. Please run PowerShell as Administrator."
    exit 1
}

$ErrorActionPreference = "Continue"
$ServiceName = "SeneschalAgent"
$InstallDir = "C:\ProgramData\Seneschal"

Write-Host "=== Seneschal Agent Uninstallation ===" -ForegroundColor Cyan

# Stop and remove service
$Service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($Service) {
    Write-Host "Stopping service..." -ForegroundColor Yellow
    sc.exe stop $ServiceName 2>$null
    Start-Sleep -Seconds 2
    
    Write-Host "Removing service..." -ForegroundColor Yellow
    sc.exe delete $ServiceName 2>$null
    Write-Host "Service removed." -ForegroundColor Green
} else {
    Write-Host "Service not found." -ForegroundColor Gray
}

# Remove installation directory
if (-not $KeepData) {
    if (Test-Path $InstallDir) {
        Write-Host "Removing installation directory: $InstallDir" -ForegroundColor Yellow
        Remove-Item -Path $InstallDir -Recurse -Force -ErrorAction SilentlyContinue
        Write-Host "Installation directory removed." -ForegroundColor Green
    }
} else {
    Write-Host "Keeping installation directory (per -KeepData flag): $InstallDir" -ForegroundColor Yellow
}

Write-Host "`n=== Uninstallation Complete ===" -ForegroundColor Cyan