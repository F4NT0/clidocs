package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── colours ───────────────────────────────────────────────────────────────────

var (
	cBg     = lipgloss.Color("#0D0D0D")
	cFg     = lipgloss.Color("#c9d1d9")
	cMuted  = lipgloss.Color("#6e7681")
	cGreen  = lipgloss.Color("#3fb950")
	cBlue   = lipgloss.Color("#58a6ff")
	cOrange = lipgloss.Color("#e8912d")
	cRed    = lipgloss.Color("#ff7b72")
	cBorder = lipgloss.Color("#30363d")
	cAccent = lipgloss.Color("#e6edf3")
	cYellow = lipgloss.Color("#d29922")
)

func sty(c lipgloss.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }

// ── ASCII art title ───────────────────────────────────────────────────────────

const asciiTitle = `
  ██████╗██╗     ██╗██████╗  ██████╗  ██████╗███████╗
 ██╔════╝██║     ██║██╔══██╗██╔═══██╗██╔════╝██╔════╝
 ██║     ██║     ██║██║  ██║██║   ██║██║     ███████╗
 ██║     ██║     ██║██║  ██║██║   ██║██║     ╚════██║
 ╚██████╗███████╗██║██████╔╝╚██████╔╝╚██████╗███████║
  ╚═════╝╚══════╝╚═╝╚═════╝  ╚═════╝  ╚═════╝╚══════╝`

// ── check result ─────────────────────────────────────────────────────────────

type checkStatus int

const (
	checkOK   checkStatus = iota
	checkWarn             // installed but outdated / non-ideal
	checkFail             // not installed
)

type checkResult struct {
	label   string
	status  checkStatus
	message string
}

// ── step enum ─────────────────────────────────────────────────────────────────

type step int

const (
	stepWelcome   step = iota // ASCII art + continue prompt
	stepChecking              // running prerequisite checks (async)
	stepChecksDone            // show results, wait for user to continue
	stepChooseDir             // pick %LOCALAPPDATA%\clidocs or custom
	stepCustomDir             // type custom path
	stepAlready               // exe already exists at dest — update?
	stepInstalling            // running doInstall async
	stepDone                  // success
	stepError                 // failure
)

// ── messages ──────────────────────────────────────────────────────────────────

type checksResultMsg struct{ results []checkResult }
type installResultMsg struct {
	dest    string
	updated bool
	err     error
}
type alreadyExistsMsg struct{}

// ── model ─────────────────────────────────────────────────────────────────────

type model struct {
	step       step
	dirChoice  int    // 0 = %LOCALAPPDATA%\clidocs, 1 = custom
	inputBuf   string // custom dir input
	destDir    string // resolved destination directory
	result     installResultMsg
	exeSrc     string
	width      int
	height     int
	checks     []checkResult
	checksFail bool // true if any check is checkFail
}

func resolveExeSrc() string {
	self, _ := os.Executable()
	candidate := filepath.Join(filepath.Dir(self), "clidocs.exe")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
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

	case checksResultMsg:
		m.checks = msg.results
		m.checksFail = false
		for _, c := range m.checks {
			if c.status == checkFail {
				m.checksFail = true
				break
			}
		}
		m.step = stepChecksDone
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
				m.step = stepChecking
				return m, runChecks()
			case "q", "ctrl+c":
				return m, tea.Quit
			}

		// ── checking (spinner placeholder — no input) ──────────────────────
		case stepChecking:
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}

		// ── checks done ───────────────────────────────────────────────────────
		case stepChecksDone:
			switch msg.String() {
			case "enter", " ":
				if m.checksFail {
					return m, tea.Quit
				}
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
func (m model) checkAndInstall() tea.Cmd {
	dest := filepath.Join(m.destDir, "clidocs.exe")
	if _, err := os.Stat(dest); err == nil {
		return func() tea.Msg { return alreadyExistsMsg{} }
	}
	return doInstall(m.exeSrc, m.destDir, false)
}

// ── prerequisite checks ───────────────────────────────────────────────────────

func runChecks() tea.Cmd {
	return func() tea.Msg {
		results := []checkResult{
			checkPowerShell(),
			checkNerdFont(),
			checkGo(),
		}
		return checksResultMsg{results: results}
	}
}

// checkPowerShell verifies pwsh (PowerShell 7+) is installed and its version.
// Minimum recommended version: 7.2 (LTS).
func checkPowerShell() checkResult {
	const minMajor = 7
	const minMinor = 2
	const label = "PowerShell"

	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		// Fallback: old Windows PowerShell 5.x
		_, err2 := exec.LookPath("powershell")
		if err2 != nil {
			return checkResult{
				label:   label,
				status:  checkFail,
				message: "PowerShell not found — install PowerShell 7+ from https://aka.ms/powershell",
			}
		}
		return checkResult{
			label:   label,
			status:  checkWarn,
			message: "Windows PowerShell 5.x found — upgrade to PowerShell 7+ (pwsh) for best experience",
		}
	}

	out, err := exec.Command(pwsh, "-NoProfile", "-NonInteractive", "-Command", "$PSVersionTable.PSVersion.ToString()").Output()
	if err != nil {
		return checkResult{label: label, status: checkWarn, message: "pwsh found but version could not be determined"}
	}
	vStr := strings.TrimSpace(string(out))
	parts := strings.Split(vStr, ".")
	major, minor := 0, 0
	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}

	if major < minMajor || (major == minMajor && minor < minMinor) {
		return checkResult{
			label:  label,
			status: checkWarn,
			message: fmt.Sprintf(
				"PowerShell %s found — update to %d.%d or newer for best compatibility",
				vStr, minMajor, minMinor,
			),
		}
	}
	return checkResult{
		label:   label,
		status:  checkOK,
		message: fmt.Sprintf("PowerShell %s is installed", vStr),
	}
}

// checkNerdFont looks for common Nerd Font families in the Windows font registry.
func checkNerdFont() checkResult {
	const label = "Nerd Font"

	// Query registry for installed fonts
	out, err := exec.Command("reg", "query",
		`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`,
	).Output()
	if err != nil {
		// Also try user fonts
		out2, err2 := exec.Command("reg", "query",
			`HKCU\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`,
		).Output()
		if err2 != nil {
			return checkResult{
				label:   label,
				status:  checkWarn,
				message: "Could not query installed fonts — ensure a Nerd Font is installed for icon support",
			}
		}
		out = out2
	} else {
		// merge user fonts too
		out2, _ := exec.Command("reg", "query",
			`HKCU\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`,
		).Output()
		out = append(out, out2...)
	}

	fontList := strings.ToLower(string(out))

	nerdPatterns := []string{
		"nerd font",
		"nerdfont",
		"fira code",
		"firacode",
		"jetbrains mono",
		"jetbrainsmono",
		"hack nerd",
		"cascadia code",
		"cascadiacode",
		"meslo",
		"source code pro",
	}

	re := regexp.MustCompile(`nerd|fira code|firacode|jetbrains mono|jetbrainsmono|hack nerd`)
	strictNerd := re.MatchString(fontList)

	for _, p := range nerdPatterns {
		if strings.Contains(fontList, p) {
			if strictNerd {
				return checkResult{
					label:   label,
					status:  checkOK,
					message: "Nerd Font detected — icons will render correctly",
				}
			}
			return checkResult{
				label:   label,
				status:  checkWarn,
				message: "Compatible font found but may not be a full Nerd Font — FiraCode Nerd Font or JetBrainsMono NF recommended",
			}
		}
	}

	return checkResult{
		label:   label,
		status:  checkWarn,
		message: "No Nerd Font detected — install FiraCode NF or JetBrainsMono NF for proper icon rendering",
	}
}

// checkGo verifies the Go toolchain is present and its version.
// Minimum recommended: 1.21.
func checkGo() checkResult {
	const label = "Go toolchain"
	const minMajor = 1
	const minMinor = 21

	goPath, err := exec.LookPath("go")
	if err != nil {
		return checkResult{
			label:   label,
			status:  checkFail,
			message: "Go not found — install from https://go.dev/dl/ to build clidocs from source",
		}
	}

	out, err := exec.Command(goPath, "version").Output()
	if err != nil {
		return checkResult{label: label, status: checkWarn, message: "Go found but version could not be determined"}
	}
	// e.g. "go version go1.24.2 windows/amd64"
	vStr := strings.TrimSpace(string(out))
	re := regexp.MustCompile(`go(\d+)\.(\d+)`)
	m := re.FindStringSubmatch(vStr)
	major, minor := 0, 0
	if len(m) == 3 {
		major, _ = strconv.Atoi(m[1])
		minor, _ = strconv.Atoi(m[2])
	}

	shortVer := ""
	if len(m) == 3 {
		shortVer = "go" + m[1] + "." + m[2]
	}

	if major < minMajor || (major == minMajor && minor < minMinor) {
		return checkResult{
			label:  label,
			status: checkWarn,
			message: fmt.Sprintf(
				"%s found — update to go%d.%d+ for full compatibility",
				shortVer, minMajor, minMinor,
			),
		}
	}
	return checkResult{
		label:   label,
		status:  checkOK,
		message: fmt.Sprintf("%s is installed", shortVer),
	}
}

// ── view ──────────────────────────────────────────────────────────────────────

func (m model) View() string {
	w := m.width
	if w < 60 {
		w = 80
	}
	boxW := w - 8
	if boxW > 88 {
		boxW = 88
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

	// Shared ASCII header rendered on every screen
	titleStyle := lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	subtitleStyle := sty(cBlue)
	mutedStyle := sty(cMuted)

	header := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(asciiTitle),
		subtitleStyle.Render("  Terminal-native snippet manager · command: clidocs"),
		mutedStyle.Render("  Organize, preview, and edit code snippets in a three-panel TUI"),
		"",
	)

	var content string

	switch m.step {

	// ── welcome ───────────────────────────────────────────────────────────────
	case stepWelcome:
		srcOk := ""
		if _, err := os.Stat(m.exeSrc); err == nil {
			srcOk = sty(cGreen).Render("  ✔ Found: ") + sty(cBlue).Render(m.exeSrc)
		} else {
			srcOk = sty(cRed).Render("  ✖ Not found: ") + mutedStyle.Render(m.exeSrc)
		}
		content = box(
			sty(cAccent).Bold(true).Render(" Welcome to the clidocs installer"),
			"",
			mutedStyle.Render("  This installer will:"),
			mutedStyle.Render("    • Copy clidocs.exe to an install directory"),
			mutedStyle.Render("    • Add that directory to your user PATH"),
			mutedStyle.Render("    • After setup, run  clidocs  in any terminal"),
			"",
			sty(cAccent).Render("  Prerequisites will be checked before installation."),
			"",
			srcOk,
			"",
			mutedStyle.Render("  Enter: continue   q: quit"),
		)

	// ── checking ──────────────────────────────────────────────────────────────
	case stepChecking:
		content = box(
			sty(cOrange).Bold(true).Render(" Checking prerequisites..."),
			"",
			mutedStyle.Render("  Please wait while your environment is verified."),
		)

	// ── checks done ───────────────────────────────────────────────────────────
	case stepChecksDone:
		rows := []string{
			sty(cAccent).Bold(true).Render(" Environment check results"),
			"",
		}
		for _, c := range m.checks {
			rows = append(rows, renderCheck(c))
		}
		rows = append(rows, "")

		if m.checksFail {
			rows = append(rows,
				sty(cRed).Render("  One or more required dependencies are missing."),
				sty(cRed).Render("  Please install them and re-run the installer."),
				"",
				mutedStyle.Render("  Enter / q: exit"),
			)
		} else {
			rows = append(rows,
				sty(cGreen).Render("  All checks passed — ready to install!"),
				"",
				mutedStyle.Render("  Enter: continue   q: quit"),
			)
		}
		content = box(rows...)

	// ── choose dir ────────────────────────────────────────────────────────────
	case stepChooseDir:
		opts := []string{
			"%LOCALAPPDATA%\\clidocs  (recommended)",
			"Custom directory",
		}
		rows := []string{
			sty(cAccent).Bold(true).Render(" Choose install directory"),
			mutedStyle.Render("  clidocs will be available as a command once the directory is in PATH."),
			"",
		}
		for i, o := range opts {
			if i == m.dirChoice {
				rows = append(rows, sty(cOrange).Render("  > ")+sty(cBlue).Bold(true).Render(o))
			} else {
				rows = append(rows, mutedStyle.Render("    "+o))
			}
		}
		rows = append(rows, "", mutedStyle.Render("  ↑↓: select   Enter: confirm   q: quit"))
		content = box(rows...)

	// ── custom dir input ──────────────────────────────────────────────────────
	case stepCustomDir:
		cur := sty(cBlue).Background(cBlue).Foreground(cBg).Render(" ")
		content = box(
			sty(cAccent).Bold(true).Render(" Custom install path"),
			"",
			mutedStyle.Render("  Type the full directory path where clidocs.exe will be placed."),
			mutedStyle.Render("  Make sure this directory is (or will be) in your PATH."),
			"",
			sty(cBlue).Render("  > ")+sty(cFg).Render(m.inputBuf)+cur,
			"",
			mutedStyle.Render("  Enter: confirm   Esc: back   Ctrl+C: quit"),
		)

	// ── already exists ────────────────────────────────────────────────────────
	case stepAlready:
		dest := filepath.Join(m.destDir, "clidocs.exe")
		content = box(
			sty(cOrange).Bold(true).Render(" clidocs is already installed"),
			"",
			mutedStyle.Render("  Found at: ")+sty(cBlue).Render(dest),
			"",
			sty(cFg).Render("  Do you want to update it to the latest version? (y/n)"),
			"",
			mutedStyle.Render("  y / Enter: update   n / Esc: cancel"),
		)

	// ── installing ────────────────────────────────────────────────────────────
	case stepInstalling:
		content = box(
			sty(cOrange).Bold(true).Render(" Installing..."),
			"",
			mutedStyle.Render("  Please wait."),
		)

	// ── done ──────────────────────────────────────────────────────────────────
	case stepDone:
		verb := "installed"
		if m.result.updated {
			verb = "updated"
		}
		content = box(
			sty(cGreen).Bold(true).Render(" ✔ clidocs "+verb+" successfully!"),
			"",
			mutedStyle.Render("  Location : ")+sty(cBlue).Render(m.result.dest),
			"",
			sty(cAccent).Render("  How to use:"),
			mutedStyle.Render("    1. Open a new terminal window (PowerShell or Windows Terminal)"),
			mutedStyle.Render("    2. Run:  clidocs"),
			mutedStyle.Render("    3. Pass a path directly:  clidocs C:\\my\\snippets"),
			"",
			mutedStyle.Render("  Press any key to exit"),
		)

	// ── error ─────────────────────────────────────────────────────────────────
	case stepError:
		content = box(
			sty(cRed).Bold(true).Render(" ✖ Installation failed"),
			"",
			sty(cRed).Render("  "+m.result.err.Error()),
			"",
			mutedStyle.Render("  Press any key to exit"),
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

// renderCheck formats a single check result with coloured badge.
func renderCheck(c checkResult) string {
	var badge string
	switch c.status {
	case checkOK:
		badge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0D0D0D")).
			Background(cGreen).
			Bold(true).
			Padding(0, 1).
			Render("OK")
	case checkWarn:
		badge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0D0D0D")).
			Background(cYellow).
			Bold(true).
			Padding(0, 1).
			Render("WARN")
	case checkFail:
		badge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Background(cRed).
			Bold(true).
			Padding(0, 1).
			Render("ERROR")
	}

	label := sty(cAccent).Bold(true).Render(c.label)
	msg := sty(cMuted).Render(c.message)
	return "  " + badge + " " + label + " — " + msg
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
	script := fmt.Sprintf(
		`$d='%s';$k='HKCU:\Environment';$p=(Get-ItemProperty $k PATH -EA SilentlyContinue).PATH;`+
			`if(-not $p){$p=$d}elseif(($p -split ';') -notcontains $d){$p=$p.TrimEnd(';')+';'+$d};`+
			`Set-ItemProperty $k PATH $p`,
		strings.ReplaceAll(dir, "'", "''"),
	)
	return runPS(script)
}

func addPowerShellAlias(exePath string) error {
	exePath = filepath.ToSlash(exePath)
	exePath = strings.ReplaceAll(exePath, "'", "''")

	script := fmt.Sprintf(`
$prof = $PROFILE.CurrentUserAllHosts
$dir  = Split-Path $prof
if (-not (Test-Path $dir))  { New-Item -ItemType Directory -Path $dir  -Force | Out-Null }
if (-not (Test-Path $prof)) { New-Item -ItemType File      -Path $prof -Force | Out-Null }

$alias = "Set-Alias -Name clidocs -Value '%s'"
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
