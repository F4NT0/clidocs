package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	repo      = "F4NT0/clidocs"
	assetName = "clidocs.exe"
)

var (
	cBlue   = lipgloss.Color("#58a6ff")
	cGreen  = lipgloss.Color("#3fb950")
	cOrange = lipgloss.Color("#e8912d")
	cFg     = lipgloss.Color("#c9d1d9")
	cMuted  = lipgloss.Color("#6e7681")
	cRed    = lipgloss.Color("#f85149")
	cNeon   = lipgloss.Color("#00ff88")
)

// Config carries values owned by the caller.
type Config struct {
	Logo    string
	Version string
}

type release struct {
	Tag    string `json:"tag_name"`
	Assets []struct {
		Name   string `json:"name"`
		URL    string `json:"browser_download_url"`
		Digest string `json:"digest"`
		Size   int64  `json:"size"`
	} `json:"assets"`
}

type state struct {
	rel        release
	url        string
	digest     string
	total      int64
	downloaded atomic.Int64
	tmp        string
	destDir    string
	destExe    string
}

type step struct {
	label, detail string
	run           func(*state) error
	real          func(*state) float64
	pct           float64
	done          bool
	err           error
}

type phase int

const (
	phConfirm phase = iota
	phRunning
	phDone
	phCancelled
)

type (
	tickMsg    struct{}
	stepDoneMs struct {
		idx int
		err error
	}
)

type model struct {
	cfg   Config
	st    *state
	steps []*step
	cur   int
	phase phase
}

func DefaultDir() string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		home, _ := os.UserHomeDir()
		local = filepath.Join(home, "AppData", "Local")
	}
	return filepath.Join(local, "clidocs")
}

// Run shows the installer TUI. skipConfirm starts immediately.
func Run(cfg Config, skipConfirm bool) error {
	st := &state{destDir: DefaultDir()}
	st.destExe = filepath.Join(st.destDir, "clidocs.exe")
	mk := func(label, detail string, run func(*state) error, real func(*state) float64) *step {
		return &step{label: label, detail: detail, run: run, real: real}
	}
	m := model{cfg: cfg, st: st, steps: []*step{
		mk("Find the latest release on GitHub", "github.com/"+repo+"/releases/latest", fetchRelease, nil),
		mk("Download clidocs.exe", "From the latest release asset", download, func(s *state) float64 {
			if s.total <= 0 {
				return 0
			}
			return float64(s.downloaded.Load()) / float64(s.total)
		}),
		mk("Verify download integrity (SHA-256)", "Compares with the digest published by GitHub", verify, nil),
		mk("Install program files", st.destDir, install, nil),
		mk("Add clidocs to the user PATH", st.destDir, func(s *state) error { return addToPath(s.destDir) }, nil),
		mk("Create the 'clidocs' command in PowerShell", "Alias in your PowerShell profile", func(s *state) error { return addAlias(s.destExe) }, nil),
	}}
	if skipConfirm {
		m.phase = phRunning
	}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return err
	}
	if fm, ok := final.(model); ok {
		os.Remove(st.tmp)
		if fm.phase == phDone && fm.failed() {
			return fmt.Errorf("installation failed")
		}
	}
	return nil
}

func (m model) failed() bool {
	for _, s := range m.steps {
		if s.err != nil {
			return true
		}
	}
	return false
}

func (m model) Init() tea.Cmd {
	if m.phase == phRunning {
		return m.start()
	}
	return nil
}

func (m model) start() tea.Cmd {
	return tea.Batch(runStep(0, m.steps[0], m.st), tick())
}

func tick() tea.Cmd {
	return tea.Tick(40*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

func runStep(i int, s *step, st *state) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(400 * time.Millisecond)
		return stepDoneMs{i, s.run(st)}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.phase {
		case phConfirm:
			switch msg.String() {
			case "y", "Y", "enter":
				m.phase = phRunning
				return m, m.start()
			case "n", "N", "esc", "q", "ctrl+c":
				m.phase = phCancelled
				return m, tea.Quit
			}
		case phDone:
			return m, tea.Quit
		case phRunning:
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		}
	case tickMsg:
		if m.phase != phRunning {
			return m, nil
		}
		s := m.steps[m.cur]
		if !s.done {
			if s.real != nil {
				s.pct = min(s.real(m.st), 0.99)
			} else if s.pct < 0.9 {
				s.pct += (0.9 - s.pct) * 0.08
			}
		}
		return m, tick()
	case stepDoneMs:
		s := m.steps[msg.idx]
		s.done, s.err, s.pct = true, msg.err, 1
		if msg.err != nil || msg.idx == len(m.steps)-1 {
			m.phase = phDone
			return m, nil
		}
		m.cur++
		return m, runStep(m.cur, m.steps[m.cur], m.st)
	}
	return m, nil
}

func (m model) View() string {
	st := func(c lipgloss.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }
	fg, muted, orange := st(cFg), st(cMuted), st(cOrange)
	green, red := st(cGreen).Bold(true), st(cRed).Bold(true)
	sep := muted.Render(strings.Repeat("─", 58))

	rows := []string{st(cNeon).Bold(true).Render(m.cfg.Logo), "", orange.Bold(true).Render("Install clidocs " + m.cfg.Version), sep}

	if m.phase == phConfirm || m.phase == phCancelled {
		rows = append(rows, fg.Render("The latest clidocs release will be downloaded from GitHub and installed."), "", muted.Render("Steps that will be performed:"))
		for i, s := range m.steps {
			rows = append(rows, orange.Render(fmt.Sprintf(" %d. ", i+1))+fg.Render(s.label), "    "+muted.Render(s.detail))
		}
		rows = append(rows, "", sep)
		if m.phase == phCancelled {
			rows = append(rows, st(cBlue).Render("Cancelled. Nothing was changed."))
		} else {
			rows = append(rows, fg.Render("Do you want to install clidocs? ")+orange.Render("[y]")+muted.Render(" yes   ")+orange.Render("[n]")+muted.Render(" no"))
		}
		return strings.Join(rows, "\n") + "\n"
	}

	for i, s := range m.steps {
		icon, status := muted.Render("○"), muted.Render("waiting")
		switch {
		case s.err != nil:
			icon, status = red.Render("✖"), red.Render("failed")
		case s.done:
			icon, status = green.Render("✔"), green.Render("done")
		case i == m.cur:
			icon, status = orange.Render("▶"), orange.Render("working...")
		}
		rows = append(rows, fmt.Sprintf(" %s %s", icon, fg.Render(s.label)),
			"   "+barView(s)+fmt.Sprintf(" %3.0f%%  ", s.pct*100)+status)
		if s.err != nil {
			rows = append(rows, "   "+red.Render(strings.SplitN(s.err.Error(), "\n", 2)[0]))
		}
	}
	rows = append(rows, sep)
	switch {
	case m.phase == phDone && m.failed():
		rows = append(rows, red.Render("Installation did not complete.")+muted.Render("  Press any key to exit"))
	case m.phase == phDone:
		rows = append(rows, green.Render("clidocs "+m.st.rel.Tag+" installed at "+m.st.destExe),
			muted.Render("Open a new terminal and run: clidocs"), muted.Render("Press any key to exit"))
	default:
		rows = append(rows, muted.Render("Installing..."))
	}
	return strings.Join(rows, "\n") + "\n"
}

func barView(s *step) string {
	c := string(cBlue)
	switch {
	case s.err != nil:
		c = string(cRed)
	case s.done:
		c = string(cNeon)
	}
	return progress.New(progress.WithSolidFill(c), progress.WithWidth(30), progress.WithoutPercentage()).ViewAs(s.pct)
}

func fetchRelease(s *state) error {
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+repo+"/releases/latest", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "clidocs-installer")
	c := &http.Client{Timeout: 30 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub returned %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&s.rel); err != nil {
		return err
	}
	for _, a := range s.rel.Assets {
		if a.Name == assetName {
			s.url, s.digest, s.total = a.URL, a.Digest, a.Size
			return nil
		}
	}
	return fmt.Errorf("release %s has no %s asset", s.rel.Tag, assetName)
}

func download(s *state) error {
	req, _ := http.NewRequest("GET", s.url, nil)
	req.Header.Set("User-Agent", "clidocs-installer")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	if resp.ContentLength > 0 {
		s.total = resp.ContentLength
	}
	f, err := os.CreateTemp("", "clidocs-*.exe")
	if err != nil {
		return err
	}
	defer f.Close()
	s.tmp = f.Name()
	buf := make([]byte, 64*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				return err
			}
			s.downloaded.Add(int64(n))
		}
		if rerr == io.EOF {
			return nil
		}
		if rerr != nil {
			return rerr
		}
	}
}

func verify(s *state) error {
	want := strings.TrimPrefix(s.digest, "sha256:")
	if want == "" || want == s.digest {
		return nil
	}
	f, err := os.Open(s.tmp)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if got := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch (expected %s, got %s)", want, got)
	}
	return nil
}

func install(s *state) error {
	if err := os.MkdirAll(s.destDir, 0755); err != nil {
		return err
	}
	// A running exe can be renamed but not overwritten.
	old := s.destExe + ".old"
	os.Remove(old)
	if _, err := os.Stat(s.destExe); err == nil {
		if err := os.Rename(s.destExe, old); err != nil {
			return err
		}
	}
	in, err := os.Open(s.tmp)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(s.destExe)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	out.Close()
	os.Remove(old)
	return nil
}

func ps(script string) error {
	sh, err := exec.LookPath("pwsh")
	if err != nil {
		sh = "powershell"
	}
	if out, err := exec.Command(sh, "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func addToPath(dir string) error {
	return ps(fmt.Sprintf(
		`$d='%s';$k='HKCU:\Environment';$p=(Get-ItemProperty $k PATH -EA SilentlyContinue).PATH;`+
			`if(-not $p){$p=$d}elseif(($p -split ';') -notcontains $d){$p=$p.TrimEnd(';')+';'+$d};`+
			`Set-ItemProperty $k PATH $p`, strings.ReplaceAll(dir, "'", "''")))
}

func addAlias(exe string) error {
	exe = strings.ReplaceAll(filepath.ToSlash(exe), "'", "''")
	return ps(fmt.Sprintf(`
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
    [System.IO.File]::AppendAllText($prof, [System.Environment]::NewLine + $block + [System.Environment]::NewLine, [System.Text.Encoding]::UTF8)
}
`, exe))
}
