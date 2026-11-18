# Copies the built app(s) to a PocketBook mounted as a drive (USB "PC link").
#   .\scripts\install.ps1 -Drive F:                 # hello
#   .\scripts\install.ps1 -Drive F: -App launcher
#   .\scripts\install.ps1 -Drive F: -All            # every app in dist/applications
# Besides copying, it registers each app on the reader (icons, view.json entry and
# the record the Launcher reads: tools/register), so the icon is on the home screen
# as soon as the cable is unplugged. An app that is copied by hand still registers
# itself the first time it runs (internal/device/desktop).
param([Parameter(Mandatory)][string]$Drive, [string]$App = "hello", [switch]$All)
$ErrorActionPreference = "Stop"
$dist = Join-Path $PSScriptRoot "..\dist\applications" | Resolve-Path
$target = Join-Path ($Drive.TrimEnd('\') + '\') "applications"
$names = if ($All) { @(Get-ChildItem $dist -Filter *.app | ForEach-Object { $_.BaseName }) } else { @($App) }
foreach ($n in $names) {
    Copy-Item (Join-Path $dist "$n.app" | Resolve-Path) $target -Force
    Write-Host "Copied $n.app to $target."
    Push-Location (Join-Path $PSScriptRoot "..")
    try { go run ./tools/register -drive $Drive -app $n; if ($LASTEXITCODE) { throw "could not register $n" } } finally { Pop-Location }
}
Write-Host "Eject safely and unplug the cable: the icons appear on the home screen."
