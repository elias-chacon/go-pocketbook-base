# Copies the built app(s) to a PocketBook mounted as a drive (USB "PC link").
#   .\scripts\install.ps1 -Drive F:                 # hello
#   .\scripts\install.ps1 -Drive F: -App launcher
#   .\scripts\install.ps1 -Drive F: -All            # every app in dist/applications
# That is the whole installation: on its first run the app registers its own
# icon on the desktop (see internal/device/desktop).
param([Parameter(Mandatory)][string]$Drive, [string]$App = "hello", [switch]$All)
$ErrorActionPreference = "Stop"
$dist = Join-Path $PSScriptRoot "..\dist\applications" | Resolve-Path
$target = Join-Path ($Drive.TrimEnd('\') + '\') "applications"
$names = if ($All) { @(Get-ChildItem $dist -Filter *.app | ForEach-Object { $_.BaseName }) } else { @($App) }
foreach ($n in $names) {
    Copy-Item (Join-Path $dist "$n.app" | Resolve-Path) $target -Force
    Write-Host "Copied $n.app to $target."
}
Write-Host "Eject safely and open it from Applications."
