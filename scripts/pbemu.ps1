<#
.SYNOPSIS
  The pbemu emulator of the PB741 firmware, driven from Windows through WSL (see docs/en/19-emulator.md).
.DESCRIPTION
  .\scripts\pbemu.ps1 setup                     clone pbemu, download the firmware, build (once)
  .\scripts\pbemu.ps1 start [-Reset]            start the reader (keeps a WSL session open: WSL stops idle containers)
  .\scripts\pbemu.ps1 stop                      stop it
  .\scripts\pbemu.ps1 install [app ...]         copy dist/applications/<app>.app into the emulated reader (all when none)
  .\scripts\pbemu.ps1 shot out.png              the screen as a PNG
  .\scripts\pbemu.ps1 tap X Y                   touch (device pixels, 1404 x 1872)
  .\scripts\pbemu.ps1 swipe X1 Y1 X2 Y2 [N MS]  drag in N steps, MS milliseconds apart
  .\scripts\pbemu.ps1 log <app>                 the app's log files of the emulated reader (applications/.pbgo)
  .\scripts\pbemu.ps1 crash                     the last Go panic the firmware's monitor log holds
  The first run of an app registers its icon: stop and start the emulator, then open Apps on the home screen.
#>
param(
    [Parameter(Mandatory, Position = 0)][ValidateSet("setup", "start", "stop", "install", "shot", "tap", "swipe", "log", "crash")][string]$Command,
    [Parameter(ValueFromRemainingArguments)][string[]]$Rest,
    [switch]$Reset
)
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$wslRoot = (wsl -e wslpath -a $root).Trim()
$work = '$HOME/pbemu-work'
$fw = "U741_6.8.4079"
$live = "$work/pbemu/$fw/.live"
$podman = "$work/bin/podman"

function Invoke-Wsl([string]$script) { wsl -e sh -c $script; if ($LASTEXITCODE) { throw "wsl exited with $LASTEXITCODE" } }

switch ($Command) {
    "setup" { Invoke-Wsl "sh '$wslRoot/scripts/pbemu/setup.sh' < /dev/null" }
    "start" {
        # WSL shuts its VM down when no session is open, and the containers die with it.
        Start-Process -WindowStyle Hidden wsl -ArgumentList "-e", "sleep", "2147483647"
        $reset = if ($Reset) { "--reset" } else { "" }
        Invoke-Wsl "cd $work/pbemu && PODMAN=$podman ./pbemu stop >/dev/null 2>&1 </dev/null; PODMAN=$podman timeout 280 ./pbemu start $reset --no-viewer --no-audio $fw </dev/null"
        Start-Sleep 15
    }
    "stop" { Invoke-Wsl "cd $work/pbemu && PODMAN=$podman ./pbemu stop </dev/null" }
    "install" {
        $apps = if ($Rest) { $Rest } else { (Get-ChildItem "$root\dist\applications\*.app").BaseName }
        foreach ($a in $apps) { Invoke-Wsl "python3 $work/pbx.py install '$wslRoot/dist/applications/$a.app'" }
    }
    "shot" {
        $out = (wsl -e wslpath -a ([IO.Path]::GetFullPath($Rest[0]))).Trim()
        Invoke-Wsl "python3 $work/pbx.py shot '$out'"
    }
    "tap" { Invoke-Wsl "python3 $work/pbx.py tap $($Rest -join ' ')" }
    "swipe" { Invoke-Wsl "python3 $work/pbx.py swipe $($Rest -join ' ')" }
    "log" { Invoke-Wsl "cd $live/mnt/ext1/applications && for f in .pbgo/$($Rest[0]).* $($Rest[0]).log; do [ -f `"`$f`" ] && { echo `"== `$f`"; tail -n 60 `"`$f`"; }; done; true" }
    "crash" { Invoke-Wsl "grep -n -B1 -A14 '^panic:' $live/var/log/monitor.log | tail -40; true" }
}
