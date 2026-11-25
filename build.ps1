# Windows equivalent of `make dist`. Uses Docker Desktop, or Docker inside WSL if
# `docker` is not on the Windows PATH. Every build increments the BUILD number of .version (the
# release); each app has its own apps/<id>/.version, bumped only when that app's sources changed
# (tools/bumpversion), and embedded in its binary (see internal/kernel/version).
#   .\build.ps1                 # builds ./apps/hello
#   .\build.ps1 -App launcher   # builds ./apps/launcher
#   .\build.ps1 -All            # builds every folder of ./apps (one release bump; an app keeps its version unless it changed)
param([string]$App = "hello", [switch]$All, [string]$Image = "go-pocketbook-base-sdk")
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot   # always build from the project root

# Only a real docker executable counts: a PowerShell function/alias named `docker`
# (e.g. a profile shim that calls `wsl docker`) may not forward arguments.
if (Get-Command docker -CommandType Application -ErrorAction SilentlyContinue) {
    $dk = @("docker"); $src = "$PWD"
} else {
    $dk = @("wsl", "-e", "docker"); $src = (wsl -e wslpath -a "$PWD").Trim()
}
Write-Host "Docker: $($dk -join ' ')   project: $src"

function Dk {
    $a = @($dk | Select-Object -Skip 1) + $args
    Write-Host "> $($dk[0]) $($a -join ' ')" -ForegroundColor DarkGray
    & $dk[0] @a
    if ($LASTEXITCODE) { throw "docker exited with code $LASTEXITCODE" }
}

$apps = if ($All) { @(Get-ChildItem apps -Directory | ForEach-Object { $_.Name }) } else { @($App) }
foreach ($a in $apps) { Remove-Item "dist/applications/$a.app" -ErrorAction SilentlyContinue }   # never report a stale file as success

$version = (go run ./tools/bumpversion build).Trim()
if ($LASTEXITCODE -or -not $version) { throw "could not bump the version" }
Write-Host "Version: $version"

Dk build -t $Image docker
foreach ($a in $apps) {
    $out = "dist/applications/$a.app"
    $own = (go run ./tools/bumpversion -app $a auto).Trim()
    if ($LASTEXITCODE -or -not $own) { throw "could not version $a" }
    Dk run --rm -v "${src}:/app" -v pbgo-cache:/gopath $Image build -trimpath "-ldflags=-s -w -buildid= -X=go-pocketbook-base/internal/kernel/version.Value=$own" -o $out "./apps/$a"
    if (-not (Test-Path $out)) { throw "Build finished but $out was not created (see the docker output above)." }
    Write-Host "Done: $a v$own (release v$version) $out ($([math]::Round((Get-Item $out).Length/1MB,1)) MB) - copy this single file to <device>/applications (USB)."
}
