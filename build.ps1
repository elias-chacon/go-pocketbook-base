# Windows equivalent of `make dist`. Uses Docker Desktop, or Docker inside WSL if
# `docker` is not on the Windows PATH.
param([string]$App = "hello", [string]$Image = "go-pocketbook-base-sdk")
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

$out = "dist/applications/$App.app"
Remove-Item $out -ErrorAction SilentlyContinue   # never report a stale file as success

Dk build -t $Image docker
Dk run --rm -v "${src}:/app" -v pbgo-cache:/gopath $Image build -o $out "./cmd/$App"

if (-not (Test-Path $out)) { throw "Build finished but $out was not created (see the docker output above)." }
Write-Host "Done: $out ($([math]::Round((Get-Item $out).Length/1MB,1)) MB) - copy this single file to <device>/applications (USB)."
