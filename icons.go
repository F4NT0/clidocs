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

// Icons are the extension label shown to the left of the filename.
// Colors follow the language's official/community color.
var extMap = map[string]langInfo{
	".go":         {icon: "go",     color: "#00add8"}, // Go cyan
	".js":         {icon: "js",     color: "#f0db4f"}, // JS yellow
	".ts":         {icon: "ts",     color: "#3178c6"}, // TS blue
	".tsx":        {icon: "tsx",    color: "#61dafb"}, // React light blue
	".jsx":        {icon: "jsx",    color: "#61dafb"},
	".py":         {icon: "py",     color: "#4b9cd3"}, // Python light blue
	".rs":         {icon: "rs",     color: "#ce422b"}, // Rust orange-red
	".sh":         {icon: "sh",     color: "#89e051"}, // Shell light green
	".bash":       {icon: "sh",     color: "#89e051"},
	".zsh":        {icon: "zsh",    color: "#89e051"},
	".ps1":        {icon: "ps1",    color: "#5391fe"}, // PowerShell medium blue
	".psm1":       {icon: "psm",    color: "#5391fe"},
	".psd1":       {icon: "psd",    color: "#5391fe"},
	".json":       {icon: "json",   color: "#f1c40f"}, // JSON yellow
	".yaml":       {icon: "yml",    color: "#f1c40f"}, // YAML yellow
	".yml":        {icon: "yml",    color: "#f1c40f"},
	".toml":       {icon: "toml",   color: "#e67e22"}, // TOML orange
	".md":         {icon: "md",     color: "#519aba"}, // Markdown steel blue
	".mdx":        {icon: "mdx",    color: "#519aba"},
	".html":       {icon: "html",   color: "#e34c26"}, // HTML orange-red
	".htm":        {icon: "html",   color: "#e34c26"},
	".css":        {icon: "css",    color: "#264de4"}, // CSS blue
	".scss":       {icon: "scss",   color: "#c6538c"}, // SCSS pink
	".sass":       {icon: "sass",   color: "#c6538c"},
	".c":          {icon: "c",      color: "#6e9bd1"}, // C light blue-grey
	".h":          {icon: "h",      color: "#6e9bd1"},
	".cpp":        {icon: "cpp",    color: "#f34b7d"}, // C++ pink
	".cc":         {icon: "cpp",    color: "#f34b7d"},
	".cs":         {icon: "cs",     color: "#9b4f96"}, // C# purple
	".java":       {icon: "java",   color: "#b07219"}, // Java brown-orange
	".kt":         {icon: "kt",     color: "#7f52ff"}, // Kotlin purple
	".kts":        {icon: "kts",    color: "#7f52ff"},
	".swift":      {icon: "swift",  color: "#f05138"}, // Swift orange-red
	".rb":         {icon: "rb",     color: "#cc342d"}, // Ruby red
	".php":        {icon: "php",    color: "#777bb3"}, // PHP purple-grey
	".lua":        {icon: "lua",    color: "#00007c"}, // Lua dark blue
	".vim":        {icon: "vim",    color: "#199f4b"}, // Vim green
	".sql":        {icon: "sql",    color: "#e38c00"}, // SQL amber
	".xml":        {icon: "xml",    color: "#0060ac"}, // XML blue
	".txt":        {icon: "txt",    color: "#6e7681"}, // plain muted
	".conf":       {icon: "conf",   color: "#6e7681"},
	".cfg":        {icon: "cfg",    color: "#6e7681"},
	".ini":        {icon: "ini",    color: "#6e7681"},
	".env":        {icon: "env",    color: "#ecd53f"}, // dotenv yellow
	".r":          {icon: "r",      color: "#198ce7"}, // R blue
	".ex":         {icon: "ex",     color: "#6e4a7e"}, // Elixir purple
	".exs":        {icon: "exs",    color: "#6e4a7e"},
	".dart":       {icon: "dart",   color: "#00b4ab"}, // Dart teal
	".vue":        {icon: "vue",    color: "#42b883"}, // Vue green
	".svelte":     {icon: "svelte", color: "#ff3e00"}, // Svelte orange
	".tf":         {icon: "tf",     color: "#7b42bc"}, // Terraform purple
	".hcl":        {icon: "hcl",    color: "#7b42bc"},
	".dockerfile": {icon: "dock",   color: "#0db7ed"}, // Docker light blue
}

var nameMap = map[string]langInfo{
	"dockerfile":  {icon: "dock", color: "#0db7ed"},
	"makefile":    {icon: "make", color: "#6d8086"},
	"jenkinsfile": {icon: "jenk", color: "#d24939"},
	".gitignore":  {icon: "git",  color: "#f05033"},
	".gitconfig":  {icon: "git",  color: "#f05033"},
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
	// fallback: use raw extension without dot, muted color
	if ext != "" {
		return strings.TrimPrefix(ext, "."), "#6e7681"
	}
	return "file", "#6e7681"
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
