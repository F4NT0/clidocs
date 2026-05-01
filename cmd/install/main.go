package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── colours ───────────────────────────────────────────────────────────────────

var (
	cBg     = lipgloss.Color("#0d1117")
	cFg     = lipgloss.Color("#c9d1d9")
	cMuted  = lipgloss.Color("#6e7681")
	cGreen  = lipgloss.Color("#3fb950")
	cBlue   = lipgloss.Color("#58a6ff")
	cOrange = lipgloss.Color("#e8912d")
	cRed    = lipgloss.Color("#ff7b72")
	cBorder = lipgloss.Color("#30363d")
	cAccent = lipgloss.Color("#e6edf3")
)

func sty(c lipgloss.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }

// ── step enum ─────────────────────────────────────────────────────────────────

type step int

const (
	stepWelcome   step = iota // show source exe + continue prompt
	stepChooseDir             // pick %LOCALAPPDATA%\clidocs or custom
	stepCustomDir             // type custom path
	stepAlready               // exe already exists at dest — update?
	stepInstalling            // running doInstall async
	stepDone                  // success
	stepError                 // failure
)

// ── messages ──────────────────────────────────────────────────────────────────

type installResultMsg struct {
	dest    string
	updated bool
	err     error
}

type alreadyExistsMsg struct{}

// ── model ─────────────────────────────────────────────────────────────────────

type model struct {
	step      step
	dirChoice int    // 0 = %LOCALAPPDATA%\clidocs, 1 = custom
	inputBuf  string // custom dir input
	destDir   string // resolved destination directory
	result    installResultMsg
	exeSrc    string
	width     int
	height    int
}

func resolveExeSrc() string {
	// prefer sibling of the installer binary
	self, _ := os.Executable()
	candidate := filepath.Join(filepath.Dir(self), "clidocs.exe")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	// fallback: cwd
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "clidocs.exe")
}

func defaultDestDir() string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		home, _ := os.UserHomeDir()
		local = filepath.Join(home, "AppData", "Local")
	}
	return filepath.Join(local, "clidocs")
}

func newModel() model {
	return model{
		step:      stepWelcome,
		dirChoice: 0,
		exeSrc:    resolveExeSrc(),
	}
}

func (m model) Init() tea.Cmd { return nil }

// ── update ────────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case installResultMsg:
		m.result = msg
		if msg.err != nil {
			m.step = stepError
		} else {
			m.step = stepDone
		}
		return m, nil

	case alreadyExistsMsg:
		m.step = stepAlready
		return m, nil

	case tea.KeyMsg:
		switch m.step {

		// ── welcome ───────────────────────────────────────────────────────────
		case stepWelcome:
			switch msg.String() {
			case "enter", " ":
				m.step = stepChooseDir
			case "q", "ctrl+c":
				return m, tea.Quit
			}

		// ── choose dir ────────────────────────────────────────────────────────
		case stepChooseDir:
			switch msg.String() {
			case "up", "k":
				if m.dirChoice > 0 {
					m.dirChoice--
				}
			case "down", "j":
				if m.dirChoice < 1 {
					m.dirChoice++
				}
			case "enter":
				if m.dirChoice == 1 {
					m.inputBuf = ""
					m.step = stepCustomDir
				} else {
					m.destDir = defaultDestDir()
					return m, m.checkAndInstall()
				}
			case "esc", "q", "ctrl+c":
				return m, tea.Quit
			}

		// ── custom dir input ──────────────────────────────────────────────────
		case stepCustomDir:
			switch msg.String() {
			case "enter":
				d := strings.TrimSpace(m.inputBuf)
				if d != "" {
					m.destDir = d
					return m, m.checkAndInstall()
				}
			case "esc":
				m.step = stepChooseDir
			case "ctrl+c":
				return m, tea.Quit
			case "backspace", "ctrl+h":
				if len(m.inputBuf) > 0 {
					m.inputBuf = m.inputBuf[:len(m.inputBuf)-1]
				}
			default:
				if len(msg.String()) == 1 && msg.String()[0] >= 0x20 {
					m.inputBuf += msg.String()
				}
			}

		// ── already exists: ask y/n ───────────────────────────────────────────
		case stepAlready:
			switch strings.ToLower(msg.String()) {
			case "y", "enter":
				m.step = stepInstalling
				return m, doInstall(m.exeSrc, m.destDir, true)
			case "n", "esc", "q":
				return m, tea.Quit
			case "ctrl+c":
				return m, tea.Quit
			}

		// ── done / error ──────────────────────────────────────────────────────
		case stepDone, stepError:
			return m, tea.Quit
		}
	}

	return m, nil
}

// checkAndInstall checks if clidocs.exe already exists at the destination.
// If it does, sends alreadyExistsMsg to prompt the user; otherwise installs.
func (m model) checkAndInstall() tea.Cmd {
	dest := filepath.Join(m.destDir, "clidocs.exe")
	if _, err := os.Stat(dest); err == nil {
		return func() tea.Msg { return alreadyExistsMsg{} }
	}
	return doInstall(m.exeSrc, m.destDir, false)
}

// ── view ──────────────────────────────────────────────────────────────────────

func (m model) View() string {
	w := m.width
	if w < 60 {
		w = 80
	}
	boxW := w - 8
	if boxW > 80 {
		boxW = 80
	}

	box := func(lines ...string) string {
		inner := lipgloss.JoinVertical(lipgloss.Left, lines...)
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cBorder).
			Background(cBg).
			Padding(1, 3).
			Width(boxW).
			Render(inner)
	}

	header := lipgloss.JoinVertical(lipgloss.Left,
		sty(cAccent).Bold(true).Render("  clidocs installer"),
		sty(cMuted).Render("  Snippet manager for the terminal"),
		"",
	)

	var content string

	switch m.step {

	case stepWelcome:
		srcOk := ""
		if _, err := os.Stat(m.exeSrc); err == nil {
			srcOk = sty(cGreen).Render("  ✔ Found: ") + sty(cBlue).Render(m.exeSrc)
		} else {
			srcOk = sty(cRed).Render("  ✖ Not found: ") + sty(cMuted).Render(m.exeSrc)
		}
		content = box(
			sty(cAccent).Bold(true).Render(" Welcome"),
			"",
			sty(cMuted).Render("This installer will:"),
			sty(cMuted).Render("  • Copy clidocs.exe to an install directory"),
			sty(cMuted).Render("  • Add that directory to your user PATH"),
			sty(cMuted).Render("  • Create the 'clidoc' alias in your PowerShell profile"),
			"",
			srcOk,
			"",
			sty(cMuted).Render("  Enter: continue   q: quit"),
		)

	case stepChooseDir:
		opts := []string{
			"%LOCALAPPDATA%\\clidocs  (recommended)",
			"Custom directory",
		}
		rows := []string{sty(cAccent).Bold(true).Render(" Choose install directory"), ""}
		for i, o := range opts {
			if i == m.dirChoice {
				rows = append(rows, sty(cOrange).Render("  > ")+sty(cBlue).Bold(true).Render(o))
			} else {
				rows = append(rows, sty(cMuted).Render("    "+o))
			}
		}
		rows = append(rows, "", sty(cMuted).Render("  ↑↓: select   Enter: confirm   q: quit"))
		content = box(rows...)

	case stepCustomDir:
		cur := sty(cBlue).Background(cBlue).Foreground(cBg).Render(" ")
		content = box(
			sty(cAccent).Bold(true).Render(" Custom install path"),
			"",
			sty(cMuted).Render("  Type the full directory path:"),
			"",
			sty(cBlue).Render("  > ")+sty(cFg).Render(m.inputBuf)+cur,
			"",
			sty(cMuted).Render("  Enter: confirm   Esc: back   Ctrl+C: quit"),
		)

	case stepAlready:
		dest := filepath.Join(m.destDir, "clidocs.exe")
		content = box(
			sty(cOrange).Bold(true).Render(" clidocs is already installed"),
			"",
			sty(cMuted).Render("  Found at: ")+sty(cBlue).Render(dest),
			"",
			sty(cFg).Render("  Do you want to update it? (y/n)"),
			"",
			sty(cMuted).Render("  y / Enter: update   n / Esc: cancel"),
		)

	case stepInstalling:
		content = box(
			sty(cOrange).Bold(true).Render(" Installing..."),
			"",
			sty(cMuted).Render("  Please wait."),
		)

	case stepDone:
		verb := "installed"
		if m.result.updated {
			verb = "updated"
		}
		content = box(
			sty(cGreen).Bold(true).Render(" ✔ clidocs "+verb+" successfully!"),
			"",
			sty(cMuted).Render("  Location : ")+sty(cBlue).Render(m.result.dest),
			"",
			sty(cAccent).Render("  How to use:"),
			sty(cMuted).Render("    1. Open a new PowerShell window"),
			sty(cMuted).Render("       (or run:  . $PROFILE)"),
			sty(cMuted).Render("    2. Type:  clidoc"),
			"",
			sty(cMuted).Render("  Press any key to exit"),
		)

	case stepError:
		content = box(
			sty(cRed).Bold(true).Render(" ✖ Installation failed"),
			"",
			sty(cRed).Render("  "+m.result.err.Error()),
			"",
			sty(cMuted).Render("  Press any key to exit"),
		)
	}

	full := lipgloss.JoinVertical(lipgloss.Left, header, content)
	return lipgloss.NewStyle().
		Background(cBg).
		Foreground(cFg).
		Padding(1, 3).
		Width(w).
		Render(full)
}

// ── install cmd ───────────────────────────────────────────────────────────────

func doInstall(exeSrc, destDir string, updated bool) tea.Cmd {
	return func() tea.Msg {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return installResultMsg{err: fmt.Errorf("cannot create %s: %w", destDir, err)}
		}

		exeDest := filepath.Join(destDir, "clidocs.exe")

		if err := copyFile(exeSrc, exeDest); err != nil {
			return installResultMsg{err: fmt.Errorf("copy failed: %w", err)}
		}

		if err := addToUserPath(destDir); err != nil {
			return installResultMsg{err: fmt.Errorf("PATH update failed: %w", err)}
		}

		if err := addPowerShellAlias(exeDest); err != nil {
			return installResultMsg{err: fmt.Errorf("alias failed: %w", err)}
		}

		return installResultMsg{dest: exeDest, updated: updated}
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func addToUserPath(dir string) error {
	// Use reg.exe so we don't need to deal with PowerShell escaping for PATH.
	script := fmt.Sprintf(
		`$d='%s';$k='HKCU:\Environment';$p=(Get-ItemProperty $k PATH -EA SilentlyContinue).PATH;`+
			`if(-not $p){$p=$d}elseif(($p -split ';') -notcontains $d){$p=$p.TrimEnd(';')+';'+$d};`+
			`Set-ItemProperty $k PATH $p`,
		strings.ReplaceAll(dir, "'", "''"),
	)
	return runPS(script)
}

func addPowerShellAlias(exePath string) error {
	// Write alias using a here-string written to a temp file to avoid all
	// quoting/backtick issues inside Go raw strings.
	exePath = filepath.ToSlash(exePath)
	exePath = strings.ReplaceAll(exePath, "'", "''") // escape single quotes

	script := fmt.Sprintf(`
$prof = $PROFILE.CurrentUserAllHosts
$dir  = Split-Path $prof
if (-not (Test-Path $dir))  { New-Item -ItemType Directory -Path $dir  -Force | Out-Null }
if (-not (Test-Path $prof)) { New-Item -ItemType File      -Path $prof -Force | Out-Null }

$alias = "Set-Alias -Name clidoc -Value '%s'"
$start = "# >>> clidocs >>>"
$end   = "# <<< clidocs <<<"
$block = $start + [System.Environment]::NewLine + $alias + [System.Environment]::NewLine + $end

$text = [System.IO.File]::ReadAllText($prof)
if ($text -match [regex]::Escape($start)) {
    $text = [regex]::Replace($text, "(?s)" + [regex]::Escape($start) + ".*?" + [regex]::Escape($end), $block)
    [System.IO.File]::WriteAllText($prof, $text, [System.Text.Encoding]::UTF8)
} else {
    $append = [System.Environment]::NewLine + $block + [System.Environment]::NewLine
    [System.IO.File]::AppendAllText($prof, $append, [System.Text.Encoding]::UTF8)
}
`, exePath)

	return runPS(script)
}

func runPS(script string) error {
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		pwsh = "powershell"
	}
	out, err := exec.Command(pwsh, "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
