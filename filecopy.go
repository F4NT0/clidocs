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

// openDirPicker shows a modern Windows Explorer-style folder picker via PowerShell COM.
// Returns the selected directory path, or "" if cancelled.
func openDirPicker() (string, error) {
	ps := `
$code = @'
using System;
using System.Runtime.InteropServices;
using System.Runtime.InteropServices.ComTypes;

[ComImport, Guid("DC1C5A9C-E88A-4dde-A5A1-60F82A20AEF7"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IFileOpenDialog {
    [PreserveSig] int Show(IntPtr hwnd);
    void SetFileTypes(uint cFileTypes, IntPtr rgFilterSpec);
    void SetFileTypeIndex(uint iFileType);
    void GetFileTypeIndex(out uint piFileType);
    void Advise(IntPtr pfde, out uint pdwCookie);
    void Unadvise(uint dwCookie);
    void SetOptions(uint fos);
    void GetOptions(out uint pfos);
    void SetDefaultFolder(IntPtr psi);
    void SetFolder(IntPtr psi);
    void GetFolder(out IntPtr ppsi);
    void GetCurrentSelection(out IntPtr ppsi);
    void SetFileName([MarshalAs(UnmanagedType.LPWStr)] string pszName);
    void GetFileName([MarshalAs(UnmanagedType.LPWStr)] out string pszName);
    void SetTitle([MarshalAs(UnmanagedType.LPWStr)] string pszTitle);
    void SetOkButtonLabel([MarshalAs(UnmanagedType.LPWStr)] string pszText);
    void SetFileNameLabel([MarshalAs(UnmanagedType.LPWStr)] string pszLabel);
    void GetResult(out IntPtr ppsi);
    void AddPlace(IntPtr psi, int fdap);
    void SetDefaultExtension([MarshalAs(UnmanagedType.LPWStr)] string pszDefaultExtension);
    void Close(int hr);
    void SetClientGuid([In] ref Guid guid);
    void ClearClientData();
    void SetFilter(IntPtr pFilter);
    void GetResults(out IntPtr ppenum);
    void GetSelectedItems(out IntPtr ppenum);
}

[ComImport, Guid("43826D1E-E718-42EE-BC55-A1E261C37BFE"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IShellItem {
    void BindToHandler(IntPtr pbc, [In] ref Guid bhid, [In] ref Guid riid, out IntPtr ppv);
    void GetParent(out IntPtr ppsi);
    void GetDisplayName(uint sigdnName, [MarshalAs(UnmanagedType.LPWStr)] out string ppszName);
    void GetAttributes(uint sfgaoMask, out uint psfgaoAttribs);
    void Compare(IntPtr psi, uint hint, out int piOrder);
}

public class FolderPicker {
    private const uint FOS_PICKFOLDERS = 0x00000020;
    private const uint FOS_FORCEFILESYSTEM = 0x00000040;
    private const uint SIGDN_FILESYSPATH = 0x80058000;
    private static readonly Guid CLSID_FileOpenDialog = new Guid("DC1C5A9C-E88A-4dde-A5A1-60F82A20AEF7");

    public static string Pick(string title) {
        Type t = Type.GetTypeFromCLSID(CLSID_FileOpenDialog);
        IFileOpenDialog dlg = (IFileOpenDialog)Activator.CreateInstance(t);
        uint opts;
        dlg.GetOptions(out opts);
        dlg.SetOptions(opts | FOS_PICKFOLDERS | FOS_FORCEFILESYSTEM);
        dlg.SetTitle(title);
        int hr = dlg.Show(IntPtr.Zero);
        if (hr != 0) return "";
        IntPtr psi;
        dlg.GetResult(out psi);
        IShellItem si = (IShellItem)Marshal.GetObjectForIUnknown(psi);
        string path;
        si.GetDisplayName(SIGDN_FILESYSPATH, out path);
        Marshal.ReleaseComObject(si);
        Marshal.ReleaseComObject(dlg);
        return path;
    }
}
'@
Add-Type -TypeDefinition $code -Language CSharp
$result = [FolderPicker]::Pick("Select the snippets directory for clidocs")
if ($result -ne "") { Write-Output $result }
`
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		pwsh = "powershell"
	}
	cmd := exec.Command(pwsh, "-NoProfile", "-NonInteractive", "-Command", ps)
	out, err := cmd.Output()
	if err != nil {
		// fallback to old FolderBrowserDialog if COM approach fails
		return openDirPickerFallback()
	}
	raw := strings.TrimSpace(string(out))
	return raw, nil
}

// openDirPickerFallback uses the legacy FolderBrowserDialog as a safety net.
func openDirPickerFallback() (string, error) {
	ps := `
Add-Type -AssemblyName System.Windows.Forms | Out-Null
$dialog = New-Object System.Windows.Forms.FolderBrowserDialog
$dialog.Description = "Select the snippets directory for clidocs"
$dialog.ShowNewFolderButton = $true
$result = $dialog.ShowDialog()
if ($result -eq [System.Windows.Forms.DialogResult]::OK) {
    Write-Output $dialog.SelectedPath
}
`
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		pwsh = "powershell"
	}
	cmd := exec.Command(pwsh, "-NoProfile", "-NonInteractive", "-Command", ps)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("folder picker failed: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
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
