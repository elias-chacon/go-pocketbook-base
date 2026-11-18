# Copies the built app to a PocketBook mounted as a drive (USB "PC link").
#   .\scripts\install.ps1 -Drive F:
# That is the whole installation: on its first run the app registers its own
# icon on the desktop (see internal/device/desktop).
param([Parameter(Mandatory)][string]$Drive, [string]$App = "hello")
$ErrorActionPreference = "Stop"
$src = Join-Path $PSScriptRoot "..\dist\applications\$App.app" | Resolve-Path
Copy-Item $src (Join-Path ($Drive.TrimEnd('\') + '\') "applications") -Force
Write-Host "Copied $App.app to $($Drive.TrimEnd('\'))\applications. Eject safely and open it from Applications."
