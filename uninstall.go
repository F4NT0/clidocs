package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const asciiLogo = `
 ██████╗██╗     ██╗██████╗  ██████╗  ██████╗███████╗
██╔════╝██║     ██║██╔══██╗██╔═══██╗██╔════╝██╔════╝
██║     ██║     ██║██║  ██║██║   ██║██║     ███████╗
██║     ██║     ██║██║  ██║██║   ██║██║     ╚════██║
╚██████╗███████╗██║██████╔╝╚██████╔╝╚██████╗███████║
 ╚═════╝╚══════╝╚═╝╚═════╝  ╚═════╝  ╚═════╝╚══════╝`

type uninstallPhase int

const (
	phaseConfirm uninstallPhase = iota
	phaseRunning
	phaseDone
	phaseCancelled
)

type uninstallStep struct {
	label, detail string
	run           func() error
	pct           float64
	done          bool
	err           error
}

type (
	uninstallTick struct{}
	stepDoneMsg   struct {
		idx int
		err error
	}
)

type uninstallModel struct {
	phase   uninstallPhase
	steps   []*uninstallStep
	cur     int
	self    string
	dir     string
	files   []string
	dirs    []string
	started time.Time
}

func newUninstallModel(skipConfirm bool) (uninstallModel, error) {
	self, err := os.Executable()
	if err != nil {
		return uninstallModel{}, err
	}
	if r, err := filepath.EvalSymlinks(self); err == nil {
		self = r
	}
	files := findClidocsExes(self)
	var dirs []string
	seen := map[string]bool{}
	for _, f := range files {
		d := filepath.Dir(f)
		if k := strings.ToLower(d); !seen[k] {
			seen[k] = true
			dirs = append(dirs, d)
		}
	}

	mk := func(label, detail string, run func() error) *uninstallStep {
		return &uninstallStep{label: label, detail: detail, run: run}
	}
	m := uninstallModel{self: self, dir: filepath.Dir(self), files: files, dirs: dirs}
	m.steps = []*uninstallStep{
		mk("Remove the 'clidocs' command from PowerShell", "Cleans the clidocs alias block from your PowerShell profiles", removeAlias),
		mk("Remove clidocs from the user PATH", strings.Join(dirs, "\n    "), func() error { return removeFromPath(dirs) }),
		mk("Delete program files", strings.Join(files, "\n    "), func() error { return nil }),
	}
	if skipConfirm {
		m.phase = phaseRunning
	}
	return m, nil
}

func (m uninstallModel) Init() tea.Cmd {
	if m.phase == phaseRunning {
		return m.start()
	}
	return nil
}

func (m uninstallModel) start() tea.Cmd {
	return tea.Batch(runStep(0, m.steps[0]), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(40*time.Millisecond, func(time.Time) tea.Msg { return uninstallTick{} })
}

func runStep(idx int, s *uninstallStep) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(600 * time.Millisecond)
		return stepDoneMsg{idx, s.run()}
	}
}

func (m uninstallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.phase {
		case phaseConfirm:
			switch msg.String() {
			case "left", "h", "right", "l", "tab":
				// no-op: single confirm prompt
			case "y", "Y", "enter":
				m.phase = phaseRunning
				m.started = time.Now()
				return m, m.start()
			case "n", "N", "esc", "q", "ctrl+c":
				m.phase = phaseCancelled
				return m, tea.Quit
			}
		case phaseDone:
			return m, tea.Quit
		case phaseRunning:
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		}
	case uninstallTick:
		if m.phase != phaseRunning {
			return m, nil
		}
		if s := m.steps[m.cur]; !s.done && s.pct < 0.9 {
			s.pct += (0.9 - s.pct) * 0.08
		}
		return m, tickCmd()
	case stepDoneMsg:
		s := m.steps[msg.idx]
		s.done, s.err, s.pct = true, msg.err, 1
		if msg.err != nil || msg.idx == len(m.steps)-1 {
			m.phase = phaseDone
			return m, nil
		}
		m.cur++
		return m, runStep(m.cur, m.steps[m.cur])
	}
	return m, nil
}

func (m uninstallModel) failed() bool {
	for _, s := range m.steps {
		if s.err != nil {
			return true
		}
	}
	return false
}

func (m uninstallModel) View() string {
	logo := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff88")).Bold(true).Render(asciiLogo)
	title := lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render("Uninstall clidocs V" + Version)
	sep := mutedStyle.Render(strings.Repeat("─", 58))
	fg := lipgloss.NewStyle().Foreground(colorFg)
	blue := lipgloss.NewStyle().Foreground(colorAccentBlue)
	green := lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#f85149")).Bold(true)
	orange := lipgloss.NewStyle().Foreground(colorOrange)

	rows := []string{logo, "", title, sep}

	if m.phase == phaseConfirm || m.phase == phaseCancelled {
		rows = append(rows, fg.Render("This will completely remove clidocs from this computer."), "", mutedStyle.Render("Steps that will be performed:"))
		for i, s := range m.steps {
			rows = append(rows, orange.Render(fmt.Sprintf(" %d. ", i+1))+fg.Render(s.label), "    "+mutedStyle.Render(s.detail))
		}
		rows = append(rows, "", mutedStyle.Render("Your snippets are NOT deleted: "+getSnippetsDir()), sep)
		if m.phase == phaseCancelled {
			rows = append(rows, blue.Render("Cancelled. Nothing was changed."))
		} else {
			rows = append(rows, fg.Render("Do you want to uninstall clidocs? ")+orange.Render("[y]")+mutedStyle.Render(" yes   ")+orange.Render("[n]")+mutedStyle.Render(" no"))
		}
		return strings.Join(rows, "\n") + "\n"
	}

	for i, s := range m.steps {
		icon := mutedStyle.Render("○")
		switch {
		case s.err != nil:
			icon = red.Render("✖")
		case s.done:
			icon = green.Render("✔")
		case i == m.cur:
			icon = orange.Render("▶")
		}
		status := mutedStyle.Render("waiting")
		switch {
		case s.err != nil:
			status = red.Render("failed")
		case s.done:
			status = green.Render("removed")
		case i == m.cur:
			status = orange.Render("removing...")
		}
		rows = append(rows,
			fmt.Sprintf(" %s %s", icon, fg.Render(s.label)),
			"   "+uninstallBar(s)+fmt.Sprintf(" %3.0f%%  ", s.pct*100)+status)
		if s.err != nil {
			rows = append(rows, "   "+red.Render(strings.SplitN(s.err.Error(), "\n", 2)[0]))
		}
	}
	rows = append(rows, sep)
	switch {
	case m.phase == phaseDone && m.failed():
		rows = append(rows, red.Render("Uninstall did not complete.")+mutedStyle.Render("  Press any key to exit"))
	case m.phase == phaseDone:
		rows = append(rows, green.Render("clidocs was uninstalled.")+mutedStyle.Render(" Restart your terminal to refresh PATH and aliases."), mutedStyle.Render("Press any key to exit"))
	default:
		rows = append(rows, mutedStyle.Render("Removing files..."))
	}
	return strings.Join(rows, "\n") + "\n"
}

func uninstallBar(s *uninstallStep) string {
	c := string(colorAccentBlue)
	switch {
	case s.err != nil:
		c = "#f85149"
	case s.done:
		c = "#00ff88"
	}
	return progress.New(progress.WithSolidFill(c), progress.WithWidth(30), progress.WithoutPercentage()).ViewAs(s.pct)
}

// findClidocsExes returns the running exe, the default install location and every clidocs.exe found on PATH.
func findClidocsExes(self string) []string {
	cands := []string{self}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		cands = append(cands, filepath.Join(local, "clidocs", "clidocs.exe"))
	}
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		if d != "" {
			cands = append(cands, filepath.Join(d, "clidocs.exe"))
		}
	}
	if out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		`[Environment]::GetEnvironmentVariable('Path','User') -split ';' | ForEach-Object { if($_){ Join-Path $_ 'clidocs.exe' } }`).Output(); err == nil {
		for _, l := range strings.Split(string(out), "\n") {
			if l = strings.TrimSpace(l); l != "" {
				cands = append(cands, l)
			}
		}
	}
	var res []string
	seen := map[string]bool{}
	for _, c := range cands {
		if _, err := os.Stat(c); err != nil {
			continue
		}
		if r, err := filepath.EvalSymlinks(c); err == nil {
			c = r
		}
		if k := strings.ToLower(c); !seen[k] {
			seen[k] = true
			res = append(res, c)
		}
	}
	return res
}

func runUninstall(args []string) {
	skip := false
	for _, a := range args {
		if a == "-y" || a == "--yes" {
			skip = true
		}
	}
	m, err := newUninstallModel(skip)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot locate clidocs executable: %v\n", err)
		os.Exit(1)
	}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if fm, ok := final.(uninstallModel); ok && fm.phase == phaseDone && !fm.failed() {
		if err := scheduleDelete(fm.files, fm.dirs); err != nil {
			fmt.Fprintf(os.Stderr, "Could not delete %s automatically: %v\n", fm.self, err)
			os.Exit(1)
		}
	}
}

func runPowerShell(script string) error {
	ps, err := exec.LookPath("pwsh")
	if err != nil {
		ps = "powershell"
	}
	if out, err := exec.Command(ps, "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func removeAlias() error {
	return runPowerShell(`
$profiles = @($PROFILE.CurrentUserAllHosts)
foreach ($p in @('pwsh','powershell')) {
  $exe = Get-Command $p -ErrorAction SilentlyContinue
  if ($exe) { $x = & $exe.Source -NoProfile -Command '$PROFILE.CurrentUserAllHosts' 2>$null; if ($x) { $profiles += $x } }
}
foreach ($prof in ($profiles | Select-Object -Unique)) {
  if (Test-Path $prof) {
    $t = [System.IO.File]::ReadAllText($prof)
    $n = [regex]::Replace($t, "(?s)\r?\n?# >>> clidocs >>>.*?# <<< clidocs <<<\r?\n?", [System.Environment]::NewLine)
    if ($n -ne $t) { [System.IO.File]::WriteAllText($prof, $n, [System.Text.Encoding]::UTF8) }
  }
}`)
}

func psList(items []string) string {
	q := make([]string, len(items))
	for i, s := range items {
		q[i] = "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
	return "@(" + strings.Join(q, ",") + ")"
}

func removeFromPath(dirs []string) error {
	return runPowerShell(fmt.Sprintf(`
$rm = %s | ForEach-Object { $_.TrimEnd('\').ToLower() }; $k='HKCU:\Environment'
$p=(Get-ItemProperty $k PATH -EA SilentlyContinue).PATH
if ($p) { $q=(($p -split ';') | Where-Object { $_ -and ($rm -notcontains $_.TrimEnd('\').ToLower()) }) -join ';'; Set-ItemProperty $k PATH $q }`,
		psList(dirs)))
}

// scheduleDelete hands the deletion to a detached PowerShell, since a running exe cannot delete itself on Windows.
func scheduleDelete(files, dirs []string) error {
	script := fmt.Sprintf(`foreach($f in %s){ for($i=0;$i -lt 60;$i++){ Remove-Item -LiteralPath $f -Force -ErrorAction SilentlyContinue; if(-not (Test-Path -LiteralPath $f)){break}; Start-Sleep -Milliseconds 500 } }; `+
		`foreach($d in %s){ if(-not (Get-ChildItem -LiteralPath $d -Force -ErrorAction SilentlyContinue)){ Remove-Item -LiteralPath $d -Force -ErrorAction SilentlyContinue } }`,
		psList(files), psList(dirs))
	ps, err := exec.LookPath("powershell")
	if err != nil {
		if ps, err = exec.LookPath("pwsh"); err != nil {
			return err
		}
	}
	c := exec.Command(ps, "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script)
	c.Dir = os.TempDir()
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000 | 0x00000200} // CREATE_NO_WINDOW | CREATE_NEW_PROCESS_GROUP
	return c.Start()
}
