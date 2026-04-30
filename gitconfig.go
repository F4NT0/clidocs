package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type GitConfig struct {
	RepoURL  string `json:"repo_url"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func gitConfigPath(snippetsDir string) string {
	return filepath.Join(snippetsDir, ".clidocs_git.json")
}

func loadGitConfig(snippetsDir string) (GitConfig, bool) {
	data, err := os.ReadFile(gitConfigPath(snippetsDir))
	if err != nil {
		return GitConfig{}, false
	}
	var cfg GitConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return GitConfig{}, false
	}
	return cfg, cfg.RepoURL != ""
}

func saveGitConfig(snippetsDir string, cfg GitConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(gitConfigPath(snippetsDir), data, 0600)
}
