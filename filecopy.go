package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type fileCopyResultMsg struct {
	copied int
	err    error
}

// openFilePicker uses PowerShell + Windows Forms to show a native Open File dialog.
// Returns the selected file path(s), or empty string if cancelled.
func openFilePicker() ([]string, error) {
	ps := `
Add-Type -AssemblyName System.Windows.Forms | Out-Null
$dialog = New-Object System.Windows.Forms.OpenFileDialog
$dialog.Title = "Select file to copy into clidocs"
$dialog.Multiselect = $true
$dialog.Filter = "All files (*.*)|*.*"
$result = $dialog.ShowDialog()
if ($result -eq [System.Windows.Forms.DialogResult]::OK) {
    $dialog.FileNames | ForEach-Object { Write-Output $_ }
}
`
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		pwsh = "powershell"
	}
	cmd := exec.Command(pwsh, "-NoProfile", "-NonInteractive", "-Command", ps)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("file picker failed: %v", err)
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil // cancelled
	}
	lines := strings.Split(raw, "\n")
	var paths []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			paths = append(paths, l)
		}
	}
	return paths, nil
}

// copyFileToDir copies src file into destDir, preserving the filename.
// Returns an error if the destination already exists (will overwrite).
func copyFileToDir(src, destDir string) error {
	name := filepath.Base(src)
	dest := filepath.Join(destDir, name)

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("cannot open %s: %v", name, err)
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("cannot create %s: %v", name, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy failed: %v", err)
	}
	return nil
}
