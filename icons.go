package main

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type langInfo struct {
	icon  string
	color lipgloss.Color
}

var extMap = map[string]langInfo{
	".go":    {icon: "", color: "#00add8"},
	".js":    {icon: "", color: "#f0db4f"},
	".ts":    {icon: "", color: "#3178c6"},
	".tsx":   {icon: "", color: "#61dafb"},
	".jsx":   {icon: "", color: "#61dafb"},
	".py":    {icon: "", color: "#3572a5"},
	".rs":    {icon: "", color: "#dea584"},
	".sh":    {icon: "", color: "#89e051"},
	".bash":  {icon: "", color: "#89e051"},
	".ps1":   {icon: "", color: "#012456"},
	".json":  {icon: "", color: "#f1c40f"},
	".yaml":  {icon: "", color: "#cb171e"},
	".yml":   {icon: "", color: "#cb171e"},
	".toml":  {icon: "", color: "#9c4221"},
	".md":    {icon: "", color: "#519aba"},
	".html":  {icon: "", color: "#e34c26"},
	".css":   {icon: "", color: "#264de4"},
	".scss":  {icon: "", color: "#c6538c"},
	".c":     {icon: "", color: "#555555"},
	".cpp":   {icon: "", color: "#f34b7d"},
	".cs":    {icon: "󰌛", color: "#178600"},
	".java":  {icon: "", color: "#b07219"},
	".rb":    {icon: "", color: "#701516"},
	".php":   {icon: "", color: "#4f5d95"},
	".lua":   {icon: "", color: "#000080"},
	".vim":   {icon: "", color: "#199f4b"},
	".sql":   {icon: "", color: "#e38c00"},
	".xml":   {icon: "󰗀", color: "#0060ac"},
	".txt":   {icon: "", color: "#6e7681"},
	".conf":  {icon: "", color: "#6e7681"},
	".env":   {icon: "", color: "#ecd53f"},
	".dockerfile": {icon: "", color: "#0db7ed"},
	".kt":    {icon: "", color: "#7f52ff"},
	".swift": {icon: "", color: "#f05138"},
	".r":     {icon: "󰟔", color: "#198ce7"},
}

var nameMap = map[string]langInfo{
	"dockerfile": {icon: "", color: "#0db7ed"},
	"makefile":   {icon: "", color: "#427819"},
	"jenkinsfile": {icon: "", color: "#d24939"},
}

func getFileIcon(filename string) (string, lipgloss.Color) {
	lower := strings.ToLower(filename)
	if info, ok := nameMap[lower]; ok {
		return info.icon, info.color
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if info, ok := extMap[ext]; ok {
		return info.icon, info.color
	}
	return "", "#6e7681"
}

func chromaLexerForFile(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	m := map[string]string{
		".go":   "go",
		".js":   "javascript",
		".ts":   "typescript",
		".tsx":  "tsx",
		".jsx":  "jsx",
		".py":   "python",
		".rs":   "rust",
		".sh":   "bash",
		".bash": "bash",
		".ps1":  "powershell",
		".json": "json",
		".yaml": "yaml",
		".yml":  "yaml",
		".toml": "toml",
		".md":   "markdown",
		".html": "html",
		".css":  "css",
		".scss": "scss",
		".c":    "c",
		".cpp":  "cpp",
		".cs":   "csharp",
		".java": "java",
		".rb":   "ruby",
		".php":  "php",
		".lua":  "lua",
		".vim":  "vim",
		".sql":  "sql",
		".xml":  "xml",
		".kt":   "kotlin",
		".swift": "swift",
		".r":    "r",
	}
	if l, ok := m[ext]; ok {
		return l
	}
	return "text"
}
