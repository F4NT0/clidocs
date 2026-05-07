# clidocs — Installation Guide

<div align="center">

<img src="images/banner.png" alt="clidocs" width="480">

</div>

---

## Table of Contents

- [Requirements](#requirements)
- [Option A — Installer (recommended)](#option-a--installer-recommended)
- [Option B — Build from source](#option-b--build-from-source)
- [Creating a Release with binaries](#creating-a-release-with-binaries)
- [First run & configuration](#first-run--configuration)
- [Uninstall](#uninstall)

---

## Requirements

The following tools must be installed and available in `PATH` **before** running clidocs:

| Requirement | Minimum version | Why |
|---|---|---|
| **Windows 10 / 11** | 64-bit | Only supported OS |
| **PowerShell** | 5.1+ (built-in) | Used internally for file dialogs |
| **Neovim (`nvim`)** | 0.9+ | Opens snippets for editing (`Enter` / `e`) |
| **Git** | 2.x | Required for GitHub sync (`g`) |
| **Windows Terminal (`wt`)** | Any | Recommended — opens Neovim in a new tab; falls back to current terminal |
| **JetBrains Nerd Font** (or any Nerd Font) | Any | Required for folder icons (``) — set as terminal font |
| **VS Code (`code`)** *(optional)* | Any | Opens files with `v` key |

> **Go is NOT required** to run the pre-built `.exe`. It is only needed to build from source.

### Install Neovim

```powershell
winget install Neovim.Neovim
```

### Install Git

```powershell
winget install Git.Git
```

### Install Windows Terminal

```powershell
winget install Microsoft.WindowsTerminal
```

### Install a Nerd Font

1. Download [JetBrainsMono Nerd Font](https://www.nerdfonts.com/font-downloads)
2. Extract and install the `.ttf` files (right-click → Install for all users)
3. Set it as the font in Windows Terminal: **Settings → Profile → Appearance → Font face**

---

## Option A — Installer (recommended)

> This is the easiest way to install clidocs on any Windows machine.

### Steps

1. Download the latest release from the [Releases page](https://github.com/F4NT0/clidocs/releases):
   - `clidocs.exe` — the main application
   - `clidocs-install.exe` — the TUI installer

2. Place both files in the same folder (e.g. `C:\tools\clidocs\`)

3. Run the installer as **Administrator**:

```powershell
.\clidocs-install.exe
```

<!-- Screenshot placeholder -->
<!-- <img src="images/instalador-pagina-inicial.png" alt="Installer welcome screen" width="400"> -->

4. Select the installation path (default: `C:\Program Files\clidocs\`) and confirm

<!-- <img src="images/instalador-local-salvar.png" alt="Installation path" width="400"> -->

5. If a previous version exists, confirm the replacement

<!-- <img src="images/instalador-update-exe.png" alt="Update existing installation" width="400"> -->

6. On success, `clidocs` is added to the system `PATH` automatically

<!-- <img src="images/instalador-sucesso.png" alt="Installation success" width="400"> -->

7. Open a **new** PowerShell or Windows Terminal window and run:

```powershell
clidocs
```

> **Snippets directory:** `%USERPROFILE%\clidocs_snippets\` — created automatically on first run.

---

## Option B — Build from source

### Requirements (build only)

| Tool | Version |
|---|---|
| **Go** | 1.24+ |
| **Git** | 2.x |

### Steps

```powershell
# 1. Clone the repository
git clone https://github.com/F4NT0/clidocs.git
cd clidocs

# 2. Download dependencies
go mod tidy

# 3. Build the main binary
go build -o clidocs.exe .

# 4. (Optional) Build the installer
go build -o clidocs-install.exe .\installer\

# 5. Run directly
.\clidocs.exe
```

### Add to PATH manually (without the installer)

```powershell
# Run once — adds the current directory to the system PATH
$env:Path += ";$PWD"
[System.Environment]::SetEnvironmentVariable("Path", $env:Path + ";$PWD", "Machine")
```

Or copy `clidocs.exe` to a folder already in `PATH` such as `C:\Windows\System32\` (requires Administrator).

---

## Creating a Release with binaries

To produce release-ready `.exe` files:

```powershell
# Build optimised Windows binaries (stripped symbols, no console window for installer)
go build -ldflags="-s -w" -o clidocs.exe .
go build -ldflags="-s -w -H windowsgui" -o clidocs-install.exe .\installer\
```

### Publishing to GitHub Releases

```powershell
# Tag the release
git tag v1.0.0
git push origin v1.0.0
```

Then go to **GitHub → Releases → Draft a new release**, select the tag, and upload:
- `clidocs.exe`
- `clidocs-install.exe`

> Users only need to download these two files — no Go runtime or build tools required.

---

## First run & configuration

1. Launch `clidocs` — on the first run without arguments, a splash screen appears
2. Press `Enter` to use the default snippets directory (`%USERPROFILE%\clidocs_snippets\`)  
   or press `s` to browse for a custom directory
3. Press `n` in the **Folders** panel to create your first folder
4. Press `n` in the **Snippets** panel to create your first snippet
5. Press `g` to set up GitHub sync (optional)

### GitHub sync setup

Press `g` and fill in:

| Field | Example |
|---|---|
| **Repository URL** | `https://github.com/you/snippets.git` |
| **Username** | `your-github-username` |
| **Email** | `you@example.com` |

Config is saved to `.clidocs_git.json` inside the snippets directory.

---

## Uninstall

### If installed via `clidocs-install.exe`

1. Delete the installation directory (e.g. `C:\Program Files\clidocs\`)
2. Remove the directory from the system `PATH`:
   - Open **System Properties → Advanced → Environment Variables**
   - Edit `Path` under **System variables** and remove the clidocs entry

### If built from source

1. Delete `clidocs.exe` from wherever you placed it
2. Remove it from `PATH` if you added it manually
3. Optionally delete `%USERPROFILE%\clidocs_snippets\` to remove all snippets

---

<div align="center">

Made with ☕ and Go · Dark theme · Keyboard-first

Created by Gabriel Stundner

</div>
