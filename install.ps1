# Installs the latest ledger release for Windows.
#
# Usage (from PowerShell):
#   irm https://raw.githubusercontent.com/sectersion/ledger/master/install.ps1 | iex
#
# or, if you've already cloned the repo:
#   .\install.ps1

$ErrorActionPreference = "Stop"

$repo = "sectersion/ledger"
$installDir = Join-Path $env:LOCALAPPDATA "ledger"

$arch = if ([System.Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64" -or $env:PROCESSOR_IDENTIFIER -match "ARM") { "arm64" } else { "amd64" }
} else {
    "386"
}

if ($arch -eq "arm64") {
    Write-Warning "ledger's release pipeline doesn't currently publish a windows/arm64 binary — falling back to amd64 (runs fine under x64 emulation)."
    $arch = "amd64"
}
if ($arch -eq "386") {
    throw "ledger doesn't publish 32-bit Windows binaries. A 64-bit Windows install is required."
}

Write-Host "Fetching latest release info for $repo..."
$release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -Headers @{ "User-Agent" = "ledger-install-script" }

$assetName = "ledger-windows-$arch.exe"
$asset = $release.assets | Where-Object { $_.name -eq $assetName }
if (-not $asset) {
    throw "Couldn't find asset '$assetName' in release $($release.tag_name). Available assets: $($release.assets.name -join ', ')"
}

New-Item -ItemType Directory -Force -Path $installDir | Out-Null
$exePath = Join-Path $installDir "ledger.exe"

Write-Host "Downloading $($asset.name) ($($release.tag_name)) to $exePath..."
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $exePath

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    Write-Host "Adding $installDir to your user PATH..."
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    $env:Path = "$env:Path;$installDir"
    Write-Host "PATH updated. Open a new terminal for it to take effect everywhere."
} else {
    Write-Host "$installDir is already on your PATH."
}

Write-Host ""
Write-Host "Installed ledger $($release.tag_name) to $exePath"
Write-Host ""
Write-Host "Next steps:"
Write-Host "  1. Open a new terminal and run: ledger -h"
Write-Host "  2. ledger spawns the 'claude' CLI as a subprocess for every agent —"
Write-Host "     install it first if you haven't: https://docs.claude.com/claude-code"
Write-Host "     and make sure 'claude' is on your PATH (verify with: claude --version)."
Write-Host "  3. Run it against a repo: ledger run <repo> ""<task description>"""
