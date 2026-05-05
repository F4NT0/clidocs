package main

// whoamiText is the text shown when the user types "whoami" in the console.
// Edit the lines below to customize your personal info.
const whoamiText = `
  ┌─────────────────────────────────────────┐
  │  Gabriel Stundner                       │
  │  Software Developer                     │
  │                                         │
  │  GitHub  : github.com/F4NT0             │
  │  Editor  : Neovim + VS Code             │
  │  Shell   : PowerShell / pwsh            │
  │  OS      : Windows 11                   │
  │                                         │
  │  "Code is poetry written for machines"  │
  └─────────────────────────────────────────┘
`

// helpLeft and helpRight are the two columns rendered side-by-side in renderHelpConsoleModal.
const helpLeft = ` GLOBAL
  Tab/→/←   Switch panels
  s         Folders panel
  g         Sync GitHub
  G         Git config
  o         Dir info
  :         Console
  q/Ctrl+C  Quit

 FOLDERS
  ↑↓/k/j   Navigate
  Enter     Open/subfolder
  ←         Parent dir
  n         New folder
  N         New subfolder
  R         Rename
  D         Delete
  f/F       Fav / Favs modal
  H         Home dir
  /         Search folders`

const helpRight = ` SNIPPETS
  ↑↓/k/j   Navigate
  Enter     Open Neovim
  /         Search files
  n         New file
  r         Rename
  m         Move
  c         Import
  d         Delete

 PREVIEW
  ↑↓/k/j   Scroll
  /         Word search
  L         Line numbers
  c         Copy clipboard
  e         Neovim
  v         VS Code
  o         Open folder

 CONSOLE COMMANDS
  time      Work hours
  whoami    User info
  help      This help
  clear     Clear output
  exit/q    Close`
