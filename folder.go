package main

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderFavoritesModal(favorites []string, favCursor int, favDisplayLabelFunc func(string) string) string {
	title := lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render(" ★ Favorites")
	sep := mutedStyle.Render(strings.Repeat("─", 48))

	var rows []string
	rows = append(rows, title, sep)

	if len(favorites) == 0 {
		rows = append(rows, mutedStyle.Render("No favorites yet"))
	} else {
		for i, absPath := range favorites {
			label := truncate(favDisplayLabelFunc(absPath), 40)
			if i == favCursor {
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

func RenderNewFolderModal(modalInputView string) string {
	title := modalTitleStyle.Render(" New Folder")
	inputRendered := inputStyle.Width(42).Render(modalInputView)
	help := helpStyle.Render("Enter: confirm  Esc: cancel")

	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			inputRendered,
			help,
		),
	)
}

func RenderRenameFolderModal(folderOpTargetAbs, origSnippetsDir, snippetsDir, currentFolderName, modalInputView string) string {
	absPath := folderOpTargetAbs
	if absPath == "" {
		absPath = filepath.Join(snippetsDir, currentFolderName)
	}
	root := origSnippetsDir
	if root == "" {
		root = snippetsDir
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
	inputRendered := inputStyle.Width(42).Render(modalInputView)
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

func RenderDeleteFolderModal(folderOpTargetAbs, origSnippetsDir, snippetsDir, currentFolderName string) string {
	absPath := folderOpTargetAbs
	if absPath == "" {
		absPath = filepath.Join(snippetsDir, currentFolderName)
	}
	name := filepath.Base(absPath)
	// Build full relative path for display
	var fullPath string
	root := origSnippetsDir
	if root == "" {
		root = snippetsDir
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

func RenderSubfolderNavModal(subNavStack []string, subNavEntries []subNavEntry, subNavCursor int) string {
	// build breadcrumb path
	crumb := "/"
	if len(subNavStack) > 0 {
		crumb = "/" + strings.Join(subNavStack, "/")
	}
	title := modalTitleStyle.Render(" Browse Folder")
	breadcrumb := mutedStyle.Render(crumb)
	sep := mutedStyle.Render(strings.Repeat("─", 44))

	var rows []string
	rows = append(rows, title, breadcrumb, sep)

	if len(subNavEntries) == 0 {
		rows = append(rows, mutedStyle.Render("(empty)"))
	} else {
		for i, e := range subNavEntries {
			var icon string
			if e.isDir {
				icon = "󰉋 "
			} else {
				ext, extColor := getFileIcon(e.name)
				icon = lipgloss.NewStyle().Foreground(extColor).Render(ext) + " "
			}
			label := truncate(e.name, 36)
			if i == subNavCursor {
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
	if len(subNavStack) <= 1 {
		backLabel = "Backspace: close"
	}
	rows = append(rows, sep,
		helpStyle.Render("↑↓: navigate  Enter: open  "+backLabel+"  Esc: close"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func RenderNewSubfolderModal(newSubfolderParentAbs, origSnippetsDir, snippetsDir, currentFolderName, modalInputView string) string {
	var parentDisplay string
	if newSubfolderParentAbs != "" {
		root := origSnippetsDir
		if root == "" {
			root = snippetsDir
		}
		rel, err := filepath.Rel(root, newSubfolderParentAbs)
		if err != nil {
			parentDisplay = filepath.Base(newSubfolderParentAbs)
		} else {
			parentDisplay = filepath.ToSlash(rel) + "/"
		}
	} else {
		parentDisplay = currentFolderName + "/"
	}
	title := modalTitleStyle.Render(" New Subfolder")
	sep := mutedStyle.Render(strings.Repeat("─", 48))
	info := mutedStyle.Render("Inside: ") + lipgloss.NewStyle().Foreground(colorOrange).Render(parentDisplay)
	rows := []string{title, sep, info, "", " " + modalInputView, ""}
	rows = append(rows, sep, mutedStyle.Render("Enter: create  Esc: cancel"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func RenderSubfolderSelectModal(subSelectEntries []string, subSelectCursor int) string {
	title := modalTitleStyle.Render(" Select Subfolder")
	sep := mutedStyle.Render(strings.Repeat("─", 48))

	var rows []string
	rows = append(rows, title, sep)

	if len(subSelectEntries) == 0 {
		rows = append(rows, mutedStyle.Render("(no subfolders)"))
	} else {
		for i, name := range subSelectEntries {
			if i == subSelectCursor {
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

func RenderMultiDeleteConfirmModal(multiDeleteSelected map[int]bool) string {
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Bold(true).Render(" Delete Multiple Folders")
	sep := mutedStyle.Render(strings.Repeat("─", 48))

	count := len(multiDeleteSelected)
	warn := lipgloss.NewStyle().Foreground(colorFg).Render("Delete " + lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render(string(rune('0'+count))) + " folder(s) and ALL their contents?")
	alert := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Render("This cannot be undone.")
	help := helpStyle.Render("Enter / y: delete    Esc / n: cancel")

	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title, "", warn, "", alert, sep, help,
		),
	)
}

func RenderFolderSearchModal(folderSearchModalQuery string, folderSearchModalResults []folderSearchResult, folderSearchModalCursor int) string {
	title := modalTitleStyle.Render(" Search Folders")
	sep := mutedStyle.Render(strings.Repeat("─", 48))

	prompt := lipgloss.NewStyle().Foreground(colorAccentBlue).Render("/ ")
	qText := lipgloss.NewStyle().Foreground(colorFg).Render(folderSearchModalQuery)
	cur := lipgloss.NewStyle().Background(colorAccentBlue).Foreground(colorBg).Render(" ")
	searchBar := prompt + qText + cur

	var rows []string
	rows = append(rows, title, searchBar, sep)

	if len(folderSearchModalResults) == 0 {
		if folderSearchModalQuery == "" {
			rows = append(rows, mutedStyle.Render("Type to search folders..."))
		} else {
			rows = append(rows, mutedStyle.Render("No matches found"))
		}
	} else {
		const maxVisible = 12
		scrollStart := 0
		if folderSearchModalCursor >= scrollStart+maxVisible {
			scrollStart = folderSearchModalCursor - maxVisible + 1
		}

		for i := scrollStart; i < len(folderSearchModalResults) && i < scrollStart+maxVisible; i++ {
			result := folderSearchModalResults[i]
			if i == folderSearchModalCursor {
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
