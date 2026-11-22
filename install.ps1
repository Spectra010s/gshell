#!/usr/bin/env pwsh
#
# Install a prebuilt gshell release on Windows.
#
# Usage:
#   irm https://raw.githubusercontent.com/Spectra010s/gshell/v2.0.0/install.ps1 | iex
#   .\install.ps1 -Version 2.0.0
#   .\install.ps1 -InstallDir C:\Tools\gshell

[CmdletBinding()]
param(
    [string]$Version = $env:GSHELL_VERSION,
    [string]$InstallDir = $env:GSHELL_INSTALL_DIR
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$repo = 'Spectra010s/gshell'

function Fail([string]$Message) {
    Write-Error $Message
    exit 1
}

$archName = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
$arch = switch ($archName) {
    'AMD64' { 'x86_64' }
    'ARM64' { 'arm64' }
    default { Fail "unsupported architecture: $archName" }
}

if (-not $Version) {
    try {
        $headers = @{ 'User-Agent' = 'gshell-installer' }
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -Headers $headers
        $Version = $release.tag_name
    }
    catch {
        Fail "could not determine the latest release: $($_.Exception.Message)"
    }
}
$Version = $Version -replace '^v', ''

$asset = "gshell_${Version}_Windows_${arch}.zip"
$base = "https://github.com/$repo/releases/download/v$Version"
$tmp = Join-Path $env:TEMP ("gshell-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp | Out-Null

try {
    Write-Host "Downloading $asset"
    Invoke-WebRequest -Uri "$base/$asset" -OutFile (Join-Path $tmp $asset) -UseBasicParsing
    Invoke-WebRequest -Uri "$base/gshell_${Version}_checksums.txt" -OutFile (Join-Path $tmp 'checksums.txt') -UseBasicParsing

    $line = Get-Content (Join-Path $tmp 'checksums.txt') |
        Where-Object { ($_ -split '\s+')[1] -eq $asset }
    if (-not $line) { Fail "no checksum found for $asset" }
    $expected = ($line -split '\s+')[0]

    Write-Host "Verifying checksum"
    $actual = (Get-FileHash -Algorithm SHA256 -Path (Join-Path $tmp $asset)).Hash
    if ($actual -ne $expected) { Fail "checksum mismatch for $asset" }

    Expand-Archive -Path (Join-Path $tmp $asset) -DestinationPath $tmp -Force
    $exe = Join-Path $tmp 'gshell.exe'
    if (-not (Test-Path $exe)) { Fail "archive does not contain gshell.exe" }

    if (-not $InstallDir) {
        $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\gshell'
    }
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item -Path $exe -Destination (Join-Path $InstallDir 'gshell.exe') -Force
    Write-Host "Installed $(Join-Path $InstallDir 'gshell.exe') ($Version)"

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $parts = @()
    if ($userPath) { $parts = $userPath -split ';' }
    if ($parts -notcontains $InstallDir) {
        $newPath = (@($parts) + $InstallDir) -join ';'
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        Write-Host "Added $InstallDir to the user PATH; open a new terminal."
    }

    Write-Host "For PowerShell completions, run: gshell completion powershell"
}
finally {
    Remove-Item -Path $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
