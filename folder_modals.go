package main

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderFavoritesModal() string {
	title := lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render(" ★ Favorites")
	sep := mutedStyle.Render(strings.Repeat("─", 48))

	var rows []string
	rows = append(rows, title, sep)

	if len(m.favorites) == 0 {
		rows = append(rows, mutedStyle.Render("No favorites yet"))
	} else {
		for i, absPath := range m.favorites {
			label := truncate(m.favDisplayLabel(absPath), 40)
			if i == m.favCursor {
				rows = append(rows,
					arrowStyle.Render("> ")+
						lipgloss.NewStyle().Foreground(colorOrange).Render("󰉋 ")+
						lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render(label))
			} else {
				rows = append(rows,
					"   "+mutedStyle.Render("󰉋 ")+
						lipgloss.NewStyle().Foreground(colorFg).Render(label))
			}
		}
	}

	rows = append(rows, sep, helpStyle.Render("↑↓: navigate   Enter: go to folder   o: open in Explorer   f: unfavorite   Esc: close"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (m model) renderNewFolderModal() string {
	title := modalTitleStyle.Render(" New Folder")
	inputRendered := inputStyle.Width(42).Render(m.modalInput.View())
	help := helpStyle.Render("Enter: confirm  Esc: cancel")

	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			inputRendered,
			help,
		),
	)
}

func (m model) renderRenameFolderModal() string {
	// Use folderOpTargetAbs for the correct display path
	absPath := m.folderOpTargetAbs
	if absPath == "" {
		absPath = filepath.Join(m.snippetsDir, m.currentFolderName())
	}
	root := m.origSnippetsDir
	if root == "" {
		root = m.snippetsDir
	}
	var displayPath string
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		displayPath = filepath.Base(absPath)
	} else {
		displayPath = filepath.ToSlash(rel) + "/"
	}
	title := modalTitleStyle.Render(" Rename Folder")
	current := mutedStyle.Render("Current: ") + lipgloss.NewStyle().Foreground(colorAccentBlue).Render(displayPath)
	inputRendered := inputStyle.Width(42).Render(m.modalInput.View())
	help := helpStyle.Render("Enter: confirm  Esc: cancel")
	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			"",
			current,
			"",
			inputRendered,
			help,
		),
	)
}

func (m model) renderDeleteFolderModal() string {
	// Use folderOpTargetAbs (set when modal was opened) for the correct path.
	absPath := m.folderOpTargetAbs
	if absPath == "" {
		absPath = filepath.Join(m.snippetsDir, m.currentFolderName())
	}
	name := filepath.Base(absPath)
	// Build full relative path for display
	var fullPath string
	root := m.origSnippetsDir
	if root == "" {
		root = m.snippetsDir
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		fullPath = name
	} else {
		fullPath = filepath.ToSlash(rel) + "/"
	}

	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Bold(true).Render(" Delete Folder")
	sep := mutedStyle.Render(strings.Repeat("─", 48))
	warn := lipgloss.NewStyle().Foreground(colorFg).Render("Delete folder and ALL its contents?")
	nameStr := lipgloss.NewStyle().Foreground(colorAccentBlue).Bold(true).Render("  " + fullPath)
	alert := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Render("This cannot be undone.")
	help := helpStyle.Render("Enter / y: delete    Esc / n: cancel")
	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title, "", warn, nameStr, "", alert, sep, help,
		),
	)
}

func (m model) renderSubfolderNavModal() string {
	// build breadcrumb path
	crumb := "/"
	if len(m.subNavStack) > 0 {
		crumb = "/" + strings.Join(m.subNavStack, "/")
	}
	title := modalTitleStyle.Render(" Browse Folder")
	breadcrumb := mutedStyle.Render(crumb)
	sep := mutedStyle.Render(strings.Repeat("─", 44))

	var rows []string
	rows = append(rows, title, breadcrumb, sep)

	if len(m.subNavEntries) == 0 {
		rows = append(rows, mutedStyle.Render("(empty)"))
	} else {
		for i, e := range m.subNavEntries {
			var icon string
			if e.isDir {
				icon = "󰉋 "
			} else {
				ext, extColor := getFileIcon(e.name)
				icon = lipgloss.NewStyle().Foreground(extColor).Render(ext) + " "
			}
			label := truncate(e.name, 36)
			if i == m.subNavCursor {
				arrow := arrowStyle.Render("> ")
				if e.isDir {
					rows = append(rows, arrow+
						lipgloss.NewStyle().Foreground(colorAccentBlue).Render(icon)+
						lipgloss.NewStyle().Foreground(colorAccentBlue).Render(label))
				} else {
					rows = append(rows, arrow+icon+
						lipgloss.NewStyle().Foreground(colorGreen).Render(label))
				}
			} else {
				if e.isDir {
					rows = append(rows, "   "+mutedStyle.Render(icon)+
						lipgloss.NewStyle().Foreground(colorFg).Render(label))
				} else {
					rows = append(rows, "   "+icon+
						mutedStyle.Render(label))
				}
			}
		}
	}

	backLabel := "Backspace: up"
	if len(m.subNavStack) <= 1 {
		backLabel = "Backspace: close"
	}
	rows = append(rows, sep,
		helpStyle.Render("↑↓: navigate  Enter: open  "+backLabel+"  Esc: close"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (m model) renderNewSubfolderModal() string {
	// Use newSubfolderParentAbs (set when modal opened) to display the correct parent name/path.
	var parentDisplay string
	if m.newSubfolderParentAbs != "" {
		root := m.origSnippetsDir
		if root == "" {
			root = m.snippetsDir
		}
		rel, err := filepath.Rel(root, m.newSubfolderParentAbs)
		if err != nil {
			parentDisplay = filepath.Base(m.newSubfolderParentAbs)
		} else {
			parentDisplay = filepath.ToSlash(rel) + "/"
		}
	} else {
		parentDisplay = m.currentFolderName() + "/"
	}
	title := modalTitleStyle.Render(" New Subfolder")
	sep := mutedStyle.Render(strings.Repeat("─", 48))
	info := mutedStyle.Render("Inside: ") + lipgloss.NewStyle().Foreground(colorOrange).Render(parentDisplay)
	rows := []string{title, sep, info, "", " " + m.modalInput.View(), ""}
	rows = append(rows, sep, mutedStyle.Render("Enter: create  Esc: cancel"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (m model) renderSubfolderSelectModal() string {
	title := modalTitleStyle.Render(" Select Subfolder")
	sep := mutedStyle.Render(strings.Repeat("─", 48))

	var rows []string
	rows = append(rows, title, sep)

	if len(m.subSelectEntries) == 0 {
		rows = append(rows, mutedStyle.Render("(no subfolders)"))
	} else {
		for i, name := range m.subSelectEntries {
			if i == m.subSelectCursor {
				rows = append(rows,
					arrowStyle.Render("> ")+
						lipgloss.NewStyle().Foreground(colorAccentBlue).Render("󰉋 ")+
						lipgloss.NewStyle().Foreground(colorAccentBlue).Bold(true).Render(name))
			} else {
				rows = append(rows,
					"   "+mutedStyle.Render("󰉋 ")+
						lipgloss.NewStyle().Foreground(colorFg).Render(name))
			}
		}
	}

	rows = append(rows, sep,
		helpStyle.Render("↑↓: navigate   Enter: open folder   Esc: cancel"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (m model) renderMultiDeleteConfirmModal() string {
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Bold(true).Render(" Delete Multiple Folders")
	sep := mutedStyle.Render(strings.Repeat("─", 48))

	count := len(m.multiDeleteSelected)
	warn := lipgloss.NewStyle().Foreground(colorFg).Render("Delete " + lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render(string(rune('0'+count))) + " folder(s) and ALL their contents?")
	alert := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Render("This cannot be undone.")
	help := helpStyle.Render("Enter / y: delete    Esc / n: cancel")

	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title, "", warn, "", alert, sep, help,
		),
	)
}

func (m model) renderFolderSearchModal() string {
	title := modalTitleStyle.Render(" Search Folders")
	sep := mutedStyle.Render(strings.Repeat("─", 48))

	prompt := lipgloss.NewStyle().Foreground(colorAccentBlue).Render("/ ")
	qText := lipgloss.NewStyle().Foreground(colorFg).Render(m.folderSearchModalQuery)
	cur := lipgloss.NewStyle().Background(colorAccentBlue).Foreground(colorBg).Render(" ")
	searchBar := prompt + qText + cur

	var rows []string
	rows = append(rows, title, searchBar, sep)

	if len(m.folderSearchModalResults) == 0 {
		if m.folderSearchModalQuery == "" {
			rows = append(rows, mutedStyle.Render("Type to search folders..."))
		} else {
			rows = append(rows, mutedStyle.Render("No matches found"))
		}
	} else {
		const maxVisible = 12
		scrollStart := 0
		if m.folderSearchModalCursor >= scrollStart+maxVisible {
			scrollStart = m.folderSearchModalCursor - maxVisible + 1
		}

		for i := scrollStart; i < len(m.folderSearchModalResults) && i < scrollStart+maxVisible; i++ {
			result := m.folderSearchModalResults[i]
			if i == m.folderSearchModalCursor {
				arrow := arrowStyle.Render("> ")
				rows = append(rows, arrow+
					lipgloss.NewStyle().Foreground(colorAccentBlue).Render("󰉋 ")+
					lipgloss.NewStyle().Foreground(colorAccentBlue).Render(result.displayPath))
			} else {
				rows = append(rows,
					"   "+mutedStyle.Render("󰉋 ")+
						lipgloss.NewStyle().Foreground(colorFg).Render(result.displayPath))
			}
		}
	}

	rows = append(rows, sep,
		helpStyle.Render("↑↓: navigate   Enter: go to folder   Esc: cancel"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}
