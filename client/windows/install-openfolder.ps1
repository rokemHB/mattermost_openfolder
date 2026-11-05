<#
.SYNOPSIS
    Installs or removes the openfolder: URL protocol for the current user.

.DESCRIPTION
    Copies openfolder.ps1 and allowed-servers.txt to %LOCALAPPDATA%\OpenFolder
    and registers the openfolder: protocol under HKEY_CURRENT_USER.
    No administrator rights are required, and nothing outside the current
    user's profile is changed.

    Run it as the user who will click the links - NOT "as administrator",
    otherwise the protocol is registered for the administrator account.

.PARAMETER Uninstall
    Removes the registry key and the installed files.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\install-openfolder.ps1

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\install-openfolder.ps1 -Uninstall
#>
param([switch]$Uninstall)

$ErrorActionPreference = 'Stop'

$installDir = Join-Path $env:LOCALAPPDATA 'OpenFolder'
$regKey     = 'HKCU:\Software\Classes\openfolder'

if ($Uninstall) {
    Remove-Item -LiteralPath $regKey     -Recurse -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $installDir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Host 'openfolder: protocol removed.'
    return
}

$sourceHandler = Join-Path $PSScriptRoot 'openfolder.ps1'
$sourceList    = Join-Path $PSScriptRoot 'allowed-servers.txt'
foreach ($f in $sourceHandler, $sourceList) {
    if (-not (Test-Path -LiteralPath $f -PathType Leaf)) {
        throw "Missing file: $f. Keep all files of client\windows together."
    }
}

$servers = @(Get-Content -LiteralPath $sourceList |
    ForEach-Object { ($_ -replace '#.*$', '').Trim() } |
    Where-Object { $_ })
if ($servers.Count -eq 0) {
    throw 'allowed-servers.txt contains no server names. Add at least one and run again.'
}

# 1. Copy handler and allow-list
New-Item -ItemType Directory -Path $installDir -Force | Out-Null
Copy-Item -LiteralPath $sourceHandler, $sourceList -Destination $installDir -Force
$handler = Join-Path $installDir 'openfolder.ps1'

# 2. Register the protocol
$command = "powershell.exe -NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File `"$handler`" `"%1`""
New-Item -Path $regKey -Value 'URL:Open Folder' -Force | Out-Null
New-ItemProperty -Path $regKey -Name 'URL Protocol' -Value '' -PropertyType String -Force | Out-Null
New-Item -Path "$regKey\shell\open\command" -Value $command -Force | Out-Null

Write-Host 'openfolder: protocol installed.'
Write-Host "Handler:          $handler"
Write-Host "Allowed servers:  $($servers -join ', ')"
Write-Host "Test with Win+R:  openfolder://$($servers[0])/<share>/<folder>"
