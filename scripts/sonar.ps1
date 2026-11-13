# Windows wrapper: tests with coverage on Windows, then the scanner (Docker in WSL).
#   $env:SONAR_GLOBAL_TOKEN must hold a SonarQube token.
$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")
if (-not $env:SONAR_GLOBAL_TOKEN) { throw "set SONAR_GLOBAL_TOKEN to a SonarQube token" }

go test "-coverprofile=coverage.out" ./... | Out-Null
if ($LASTEXITCODE) { throw "tests failed" }

$env:WSLENV = "SONAR_GLOBAL_TOKEN/u"   # forward the token to WSL without printing it
$path = (wsl -e wslpath -a "$PWD/scripts/sonar.sh").Trim()
wsl -e bash $path
if ($LASTEXITCODE) { throw "sonar analysis failed" }
