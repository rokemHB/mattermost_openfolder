<#
.SYNOPSIS
    Protocol handler for openfolder: links.

.DESCRIPTION
    Windows calls this script with the full link, e.g.
        openfolder://fileserver.example.com/share/Projects%202026
    It converts the link to a UNC path
        \\fileserver.example.com\share\Projects 2026
    and opens it in Windows Explorer. If the link points to a file, Explorer
    opens the containing folder with that file selected (highlighted), exactly
    like a single click on it - the file itself is never opened or run.

    Safety rules (in this order, before any network access):
      1. Only UNC paths are accepted (no drive letters, no device paths).
      2. The server must be listed in allowed-servers.txt next to this script.
      3. Paths containing ".." segments or double quotes are rejected.
      4. A folder is opened, a file is only selected. If the target does not
         exist, the nearest existing parent folder on the same share is opened.

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
if ($p.Contains('"')) { exit 1 }   # cannot occur in Windows paths; keeps the argument quoting intact

if (Test-Path -LiteralPath $p -PathType Container) {
    # Folder: open it. A trailing backslash is dropped so it cannot escape the closing quote.
    Start-Process -FilePath explorer.exe -ArgumentList "`"$($p.TrimEnd('\'))`""
}
elseif (Test-Path -LiteralPath $p -PathType Leaf) {
    # File: open the containing folder with the file selected. /select never opens the file.
    Start-Process -FilePath explorer.exe -ArgumentList "/select,`"$p`""
}
else {
    # Target not found (renamed, deleted, or a link cut short at a space):
    # open the nearest existing parent folder, at most up to \\server\share.
    $q = $p.TrimEnd('\')
    while ($q -match '^\\\\[^\\]+\\[^\\]+\\.') {
        $q = $q.Substring(0, $q.LastIndexOf('\'))
        if (Test-Path -LiteralPath $q -PathType Container) {
            Start-Process -FilePath explorer.exe -ArgumentList "`"$q`""
            break
        }
    }
}
