<#
.SYNOPSIS
    Protocol handler for openfolder: links.

.DESCRIPTION
    Windows calls this script with the full link, e.g.
        openfolder://fileserver.example.com/share/Projects%202026
    It converts the link to a UNC path
        \\fileserver.example.com\share\Projects 2026
    and opens it in Windows Explorer.

    Safety rules (in this order, before any network access):
      1. Only UNC paths are accepted (no drive letters, no device paths).
      2. The server must be listed in allowed-servers.txt next to this script.
      3. Paths containing ".." segments are rejected.
      4. The target must be an existing folder. Files are never opened or run.

    Anything that fails a check is silently ignored.
#>
param([string]$Url)

$ErrorActionPreference = 'Stop'

# Allowed servers: one host name per line; empty lines and "#" comments ignored.
$listFile = Join-Path $PSScriptRoot 'allowed-servers.txt'
if (-not (Test-Path -LiteralPath $listFile -PathType Leaf)) { exit 1 }
$allowed = @(Get-Content -LiteralPath $listFile |
    ForEach-Object { ($_ -replace '#.*$', '').Trim() } |
    Where-Object { $_ })
if ($allowed.Count -eq 0) { exit 1 }

# openfolder://server/share/dir  ->  \\server\share\dir
$p = $Url -replace '^openfolder:/*', ''
$p = [System.Uri]::UnescapeDataString($p) -replace '/', '\'
$p = '\\' + $p.TrimStart('\')

# Server check before any access, so Windows never contacts another host.
if ($p -notmatch '^\\\\([^\\]+)\\[^\\]') { exit 1 }
if ($allowed -notcontains $Matches[1]) { exit 1 }   # case-insensitive
if ($p -match '\\\.\.(\\|$)') { exit 1 }

if (Test-Path -LiteralPath $p -PathType Container) {
    Start-Process -FilePath explorer.exe -ArgumentList "`"$p`""
}
