# Runs the tests of every pure-Go package (internal/**) on an emulated ARMv7, the reader's
# processor, with the SDK's Go 1.17. Nothing here needs the device or privileges:
#   stage 1  golang:1.17 compiles each package's tests for linux/arm
#   stage 2  an Alpine container runs them under qemu-arm (user mode); like the reader it has
#            no zoneinfo, which is how a test that forgot to carry the time zone database fails
# What it cannot check: InkView (it needs libinkview and the screen), the FAT filesystem of
# the reader's storage, and real speed. See docs chapter 10.
#   .\scripts\arm-test.ps1
$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")

# Only a real docker executable counts (see build.ps1).
if (Get-Command docker -CommandType Application -ErrorAction SilentlyContinue) {
    $dk = @("docker"); $src = "$PWD"; $sc = Join-Path $PWD "scripts\arm"
} else {
    $dk = @("wsl", "-e", "docker"); $src = (wsl -e wslpath -a "$PWD").Trim(); $sc = "$src/scripts/arm"
}
function Dk { $a = @($dk | Select-Object -Skip 1) + $args; & $dk[0] @a; if ($LASTEXITCODE) { throw "docker exited with code $LASTEXITCODE" } }

# The shell scripts may have been checked out with Windows line endings: strip them on the way in.
Write-Host "== compiling the tests for ARMv7 (Go 1.17)" -ForegroundColor Cyan
Dk run --rm --platform linux/amd64 -v "${src}:/src" -v "${sc}:/sc" -v pbgo-gomod:/go/pkg/mod golang:1.17 sh -c "tr -d '\r' < /sc/build.sh | sh"
Write-Host "== running them under qemu-arm" -ForegroundColor Cyan
Dk run --rm --platform linux/amd64 -v "${src}:/src" -v "${sc}:/sc" alpine:3.19 sh -c "tr -d '\r' < /sc/run.sh | sh"
