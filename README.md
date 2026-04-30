# clidocs

A TUI snippet manager for your terminal, built with Go + Bubbletea.  
Organize code snippets and notes by folder, browse them with a three-panel interface, preview with syntax highlighting, and edit directly in Neovim — all inside PowerShell on Windows 11.

---

## Requirements

- **Go 1.24+**
- **Windows 11 / PowerShell**
- **[Neovim](https://neovim.io/)** installed and available in `PATH` (required to edit files)
- A **Nerd Font** configured in your terminal (for file-type icons)

---

## Build & Run

```powershell
# Build the binary
go build -o clidocs.exe .

# Run
.\clidocs.exe
```

To add `clidocs` to your PATH permanently, copy `clidocs.exe` to any folder already in your `$env:PATH`, e.g.:

```powershell
Copy-Item .\clidocs.exe "$env:USERPROFILE\AppData\Local\Microsoft\WindowsApps\clidocs.exe"
```

---

## Snippets Directory

All snippets are stored in:

```
C:\Users\<YourUser>\clidocs_snippets\
```

The directory is created automatically on first run. Each sub-folder inside it becomes a **category** in the Folders panel.

---

## Project Structure

| File | Description |
|------|-------------|
| `main.go` | Entry point — creates snippets dir and starts the Bubbletea program |
| `dirs.go` | Resolves the path to `%USERPROFILE%\clidocs_snippets` |
| `model.go` | App state struct, folder/file loading, helper functions |
| `update.go` | All keyboard handling, modal state machine, Neovim launch via `tea.ExecProcess` |
| `view.go` | Three-panel layout renderer (Folders / Files / Preview) + modal overlay |
| `styles.go` | All Lipgloss styles using the GitHub Dark color palette |
| `icons.go` | Nerd Font icon + color mapping by file extension, Chroma lexer lookup |
| `highlight.go` | Chroma syntax highlighting (GitHub Dark theme, terminal256 formatter) |

---

## Interface Layout

```
┌─ clidocs ─────────────────────────────────────────────────────────────┐
│  [Folders]   [Snippets]   [Preview]   /  FolderName  .  filename.go   │
├──────────────┬────────────────┬───────────────────────────────────────┤
│  Folders     │  Files         │  Preview (syntax highlighted)         │
│              │                │                                       │
│  → TUI       │  6 snippets    │  package main                        │
│  Configs     │                │                                       │
│  Neovim      │   snippet.go   │  import (                            │
│  Payloads    │  TUI • 2mo ago │      "fmt"                           │
│              │                │  )                                    │
│              │   readme.md    │                                       │
│              │  TUI • 1d ago  │                                       │
├──────────────┴────────────────┴───────────────────────────────────────┤
│  Tab: switch panel  ↑↓: navigate  Enter: open/edit  n: new  q: quit  │
└───────────────────────────────────────────────────────────────────────┘
```

---

## Keybindings

| Key | Context | Action |
|-----|---------|--------|
| `Tab` / `→` | Any panel | Move focus to the next panel (right) |
| `←` | Any panel | Move focus to the previous panel (left) |
| `↑` / `k` | Folders panel | Select previous folder |
| `↓` / `j` | Folders panel | Select next folder |
| `↑` / `k` | Files panel | Select previous file |
| `↓` / `j` | Files panel | Select next file |
| `↑` / `k` | Preview panel | Scroll preview up |
| `↓` / `j` | Preview panel | Scroll preview down |
| `Enter` | Folders panel | Open folder → move focus to Files panel |
| `Enter` | Files panel | Open selected file in **Neovim** |
| `n` | Folders panel | Open **New Folder** modal |
| `n` | Files panel | Open **New File** modal (asks name + extension, then opens in Neovim) |
| `s` | Any panel | Jump back to the Folders panel |
| `g` | Any panel | Sync snippets to GitHub (opens setup on first use) |
| `G` | Any panel | Open GitHub configuration (change repo/user/email) |
| `?` | Any panel | Show keybinding hint in status bar |
| `q` / `Ctrl+C` | Any | Quit |

---

## GitHub Sync

Press `g` to sync your snippets to a GitHub repository.

### First use — Setup modal
You will be prompted for three fields (navigate with `Enter` or `Tab`):
1. **Repository URL** — e.g. `https://github.com/user/snippets.git`
2. **Username** — your GitHub username (used for git identity)
3. **Email** — your GitHub email (used for git identity)

After confirming, the config is saved to `clidocs_snippets/.clidocs_git.json` and an initial push is performed.

> **Note:** The repository must already exist on GitHub. For private repos, ensure your credentials are cached (e.g. via Git Credential Manager or SSH key).

### Change configuration
Press `G` at any time to open the configuration modal and update any field.

### Git indicator
When GitHub is configured, the header shows `  <username>` to confirm the connection.

---

## Modals

### New Folder
- Press `n` while the **Folders** panel is focused.
- Type the folder name and press `Enter` to create it.
- Press `Esc` to cancel.

### New File
- Press `n` while the **Files** panel is focused.
- **Step 1** — Type the file name (without extension), press `Enter` or `Tab`.
- **Step 2** — Type the extension (e.g. `go`, `py`, `md`), press `Enter` to create and open in Neovim.
- Press `Esc` at any step to cancel.

---

## Neovim Integration

When you press `Enter` on a file or finish creating a new file, `clidocs` launches Neovim inside the same terminal session using `tea.ExecProcess`. When you exit Neovim (`:q`), you return to `clidocs` and the preview refreshes automatically.

If Neovim is **not found in PATH**, an error modal is shown:

```
  Error

Neovim (nvim) not found in PATH.
Please install and configure Neovim to edit files.

Enter / Esc: close
```

---

## Supported Languages

Icons require a **Nerd Font**. Syntax highlighting uses **Chroma** with the **GitHub Dark** style.

| Extension | Language | Icon |
|-----------|----------|------|
| `.go` | Go | `` |
| `.js` | JavaScript | `` |
| `.ts` | TypeScript | `` |
| `.tsx` / `.jsx` | React | `` |
| `.py` | Python | `` |
| `.rs` | Rust | `` |
| `.sh` / `.bash` | Shell / Bash | `` |
| `.ps1` | PowerShell | `` |
| `.json` | JSON | `` |
| `.yaml` / `.yml` | YAML | `` |
| `.toml` | TOML | `` |
| `.md` | Markdown | `` |
| `.html` | HTML | `` |
| `.css` | CSS | `` |
| `.scss` | SCSS | `` |
| `.c` | C | `` |
| `.cpp` | C++ | `` |
| `.cs` | C# | `󰌛` |
| `.java` | Java | `` |
| `.rb` | Ruby | `` |
| `.php` | PHP | `` |
| `.lua` | Lua | `` |
| `.sql` | SQL | `` |
| `.xml` | XML | `󰗀` |
| `.kt` | Kotlin | `` |
| `.swift` | Swift | `` |
| `.r` | R | `󰟔` |
| `.txt` / `.conf` | Plain text | `` |

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/charmbracelet/bubbletea` | TUI framework (Elm architecture) |
| `github.com/charmbracelet/bubbles` | Text input component |
| `github.com/charmbracelet/lipgloss` | Layout and styling |
| `github.com/alecthomas/chroma/v2` | Syntax highlighting |
