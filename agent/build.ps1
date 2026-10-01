<#>
.SYNOPSIS
    Builds Seneschal Agent with embedded configuration defaults.

.DESCRIPTION
    Compiles agent.exe with embedded configuration defaults using -ldflags.
    The resulting binary can run without config.yaml (uses embedded defaults),
    but still allows config.yaml to override embedded values.

.PARAMETER CollectorURL
    Collector API base URL to embed (e.g., http://192.168.1.50:8081)

.PARAMETER IntervalSeconds
    Telemetry interval in seconds to embed (default: 15)

.PARAMETER DeviceType
    Device type to embed: server, pc, or laptop (default: pc)

.PARAMETER ServicesToCheck
    Comma-separated process names to embed (default: "nginx,postgres")

.PARAMETER OutputPath
    Output binary path (default: .\agent.exe)

.EXAMPLE
    .\build.ps1 -CollectorURL "http://192.168.1.50:8081" -IntervalSeconds 15 -DeviceType "pc"
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)]
    [ValidatePattern('^https?://')]
    [string]$CollectorURL,

    [Parameter()]
    [int]$IntervalSeconds = 15,

    [Parameter()]
    [ValidateSet('server','pc','laptop')]
    [string]$DeviceType = "pc",

    [Parameter()]
    [string]$ServicesToCheck = "nginx,postgres",

    [Parameter()]
    [string]$OutputPath = ".\agent.exe"
)

Write-Host "=== Building Seneschal Agent with Embedded Config ===" -ForegroundColor Cyan
Write-Host "Collector URL:  $CollectorURL" -ForegroundColor Gray
Write-Host "Interval:       ${IntervalSeconds}s" -ForegroundColor Gray
Write-Host "Device Type:    $DeviceType" -ForegroundColor Gray
Write-Host "Services:       $ServicesToCheck" -ForegroundColor Gray
Write-Host "Output:         $OutputPath" -ForegroundColor Gray

$LdFlags = @(
    "-X main.embeddedCollectorURL=$CollectorURL",
    "-X main.embeddedIntervalSeconds=$IntervalSeconds",
    "-X main.embeddedDeviceType=$DeviceType",
    "-X main.embeddedServicesToCheck=$ServicesToCheck"
) -join ' '

Write-Host "`nBuilding..." -ForegroundColor Cyan
$BuildCmd = "go build -ldflags `"$LdFlags`" -o `"$OutputPath`" ."
Write-Host "Command: $BuildCmd" -ForegroundColor DarkGray

$ExitCode = & go build -ldflags $LdFlags -o $OutputPath .
if ($LASTEXITCODE -ne 0) {
    Write-Error "Build failed with exit code $LASTEXITCODE"
    exit $LASTEXITCODE
}

Write-Host "`nBuild successful!" -ForegroundColor Green
Write-Host "Binary: $OutputPath" -ForegroundColor Gray
Write-Host "Deploy: Copy $OutputPath and config.yaml (optional) to target machine" -ForegroundColor Gray