# OpenFolder Links for Mattermost

[![CI](https://github.com/rokemHB/mattermost_openfolder/actions/workflows/ci.yml/badge.svg)](https://github.com/rokemHB/mattermost_openfolder/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Click a network path in Mattermost and it opens in Windows Explorer. Links to a file open its folder with the file selected.

Teams often share paths like `\\fileserver.example.com\projects\2026` in chat. Mattermost cannot open them: browsers and the desktop app deliberately block `file:` links from web content. This project solves that with two small parts:

| Part | Where it runs | What it does |
| --- | --- | --- |
| **Server plugin** | Mattermost server | Turns UNC paths on allowed servers into clickable `openfolder:` links when a message is posted. |
| **Protocol handler** | Each Windows client | Registers the `openfolder:` protocol and opens the folder in Explorer, after strict safety checks. For a file, it opens the containing folder and selects the file. |

## How it works

```
 User posts                       Plugin stores                    Click on Windows
 ──────────────────────────────   ──────────────────────────────   ─────────────────────────────
 \\fileserver.example.com\a b  →  [`\\fileserver...\a b`]          openfolder.ps1 checks server,
                                  (openfolder://fileserver.        path and folder, then opens
                                   example.com/a%20b)           →  \\fileserver.example.com\a b
                                                                   in Windows Explorer
```

The path stays visible exactly as typed and can still be copied. The link only works on PCs where the protocol handler is installed; everywhere else it is harmless.

## Features

- Converts bare UNC paths, paths in quotes (`"…"`, `„…“`) or backticks, and `file://server/...` URLs
- Paths with spaces in folder names and file names (`\\server\Projects 2026\Final report.docx`); text after the path stays plain text
- Trailing punctuation is trimmed, balanced brackets are kept
- Leaves code blocks, existing Markdown links and other servers untouched
- Runs before the post is saved: no bot replies, no "(edited)" marker
- Also applies when a message is edited, and upgrades links written by older versions
- Links to files open the containing folder with the file selected, without opening the file
- Allow-list of servers on both sides (plugin and client)

## Requirements

- Mattermost Server 8.0 or later, Linux (amd64 or arm64)
- Windows 10/11 clients with Windows PowerShell 5.1 (built in)
- System Admin rights in Mattermost for installation

## Installation

### 1. Install the server plugin

1. Download `openfolder-links-<version>.tar.gz` from the [latest release](https://github.com/rokemHB/mattermost_openfolder/releases/latest), or [build it yourself](#building-from-source).
2. In Mattermost, open **System Console → Plugins → Plugin Management**, click **Upload Plugin** and select the file.
3. Enable the plugin, open its settings and enter your file servers under **Allowed file servers**, for example `fileserver.example.com, nas.example.org`.

With no servers configured, the plugin changes nothing.

### 2. Allow the link scheme in Mattermost

Open **System Console → Site Configuration → Posts → Custom URL Schemes**, add `openfolder` (without colon or slashes) and save.

### 3. Install the protocol handler on each Windows client

1. Edit [`client/windows/allowed-servers.txt`](client/windows/allowed-servers.txt) and list the same servers as in the plugin settings, one per line.
2. Hand the whole `client/windows` folder to your users (the release also contains it as `openfolder-windows-client.zip`).
3. Users unzip it and double-click **`install-openfolder.cmd`**, as a normal user, *not* "Run as administrator".

No administrator rights are needed. The installer copies the handler to `%LOCALAPPDATA%\OpenFolder` and registers the protocol under `HKEY_CURRENT_USER`.

To test, press <kbd>Win</kbd>+<kbd>R</kbd> and enter `openfolder://fileserver.example.com/share/folder` with a real folder. On the first click in Mattermost, the browser or desktop app asks whether it may open the link; choose "always allow".

<details>
<summary>Deploying to many machines</summary>

- **Logon script:** run `install-openfolder.ps1` per user via Group Policy. It only writes to the user's profile.
- **Machine-wide:** copy `openfolder.ps1` and `allowed-servers.txt` to a fixed folder such as `C:\Program Files\OpenFolder\`, adapt [`openfolder.reg`](client/windows/openfolder.reg) and import it under `HKEY_LOCAL_MACHINE\SOFTWARE\Classes`. Users should not have write access to that folder.

</details>

## Security

Anyone who can post in Mattermost can create an `openfolder:` link, so the handler treats every link as untrusted input. It ignores the link unless all checks pass, in this order:

| Check | Protects against |
| --- | --- |
| Must be a UNC path `\\server\share\...` | Local paths, drive letters, device paths (`\\?\`, `\\.\`) |
| Server must be in `allowed-servers.txt` (exact host match, checked **before** any network access) | Look-alike hosts and links to external servers that could capture Windows logon hashes (NTLM) |
| No `..` path segments, no double quotes | Escaping to another share or location; breaking out of the quoted argument |
| A **folder** is opened; a **file** is only selected in its folder (`explorer /select`). If the target does not exist, the nearest existing parent folder on the same share is opened | Opening or running files, scripts or executables |

The path is passed to Explorer as a single quoted argument and is never evaluated as a command. Explorer runs with the user's own permissions, so folders the user cannot access stay inaccessible. The handler makes no other network connections and sends no data anywhere.

`-ExecutionPolicy Bypass` applies only to the single invocation of the handler and does not change any system setting. The plugin itself only rewrites message text on the server and never touches files.

Found a security issue? Please open a [private security advisory](https://github.com/rokemHB/mattermost_openfolder/security/advisories/new) instead of a public issue.

## Uninstall

- **Client:** run `install-openfolder.cmd -Uninstall` (or the `.ps1` with `-Uninstall`). This removes the registry key and `%LOCALAPPDATA%\OpenFolder`.
- **Server:** disable or remove the plugin in the System Console. Messages that were already converted keep their links.

## Building from source

Requires Go (see `server/go.mod` for the version) and `make`.

```sh
make test   # run unit tests
make dist   # build dist/openfolder-links-<version>.tar.gz
```

Pushing a tag such as `v1.0.1` builds the bundle in GitHub Actions and publishes a release with the plugin and the Windows client.

## Project layout

```
.
├── plugin.json                  Plugin manifest and settings
├── server/                      Go server plugin
│   ├── plugin.go                Hooks and configuration
│   ├── convert.go               Path detection and link conversion
│   └── convert_test.go          Unit tests
├── client/windows/              Protocol handler for Windows
│   ├── install-openfolder.cmd   Double-click installer
│   ├── install-openfolder.ps1   Installer / uninstaller
│   ├── openfolder.ps1           The handler that opens the folder or selects the file
│   ├── allowed-servers.txt      Server allow-list
│   └── openfolder.reg           Registry template for manual deployment
└── Makefile                     Test and build targets
```

## Troubleshooting

| Symptom | Cause and fix |
| --- | --- |
| Windows asks to "get an app" for this link | The handler is not installed for this user. Run `install-openfolder.cmd` again as a normal user, not as administrator. |
| Path is not turned into a link | The server is missing from the plugin settings, or the message was sent before the plugin was enabled. Editing and saving the message converts it. |
| Link is not clickable | `openfolder` is missing under Custom URL Schemes. Reload Mattermost after adding it. |
| Click does nothing | The server is missing from `allowed-servers.txt`, the share is unreachable, or the user has no access. |
| Only part of the path is linked | The last folder name contains spaces. In unquoted paths, the last part ends at the first space unless a file name follows. Put the path in quotes or backticks: `` `\\server\share\Old projects` ``. A click on the shortened link still opens the parent folder. |
| Text after the path became part of the link | The message was sent with plugin 1.0.x. Edit the message and paste the path again; plugin 1.1.0 or later links only the path. |
| A file link does nothing, folder links work | The client still runs an older handler. Run `install-openfolder.cmd` again. |
| "Encountered an error when extracting the plugin" | The bundle was repacked without directory entries. Use the release file or `make dist`. |

## License

[MIT](LICENSE)
