package main

import (
	"fmt"
	"os"
	"os/exec"
)

type gitSyncResultMsg struct {
	err    error
	output string
}

func gitSync(snippetsDir string, cfg GitConfig) (string, error) {
	gitDir := snippetsDir + string(os.PathSeparator) + ".git"
	isRepo := false
	if _, err := os.Stat(gitDir); err == nil {
		isRepo = true
	}

	run := func(args ...string) (string, error) {
		c := exec.Command("git", args...)
		c.Dir = snippetsDir
		out, err := c.CombinedOutput()
		return string(out), err
	}

	setIdentity := func() error {
		if _, err := run("config", "user.email", cfg.Email); err != nil {
			return fmt.Errorf("git config email: %v", err)
		}
		if _, err := run("config", "user.name", cfg.Username); err != nil {
			return fmt.Errorf("git config name: %v", err)
		}
		return nil
	}

	// Ensure .gitignore excludes the config file
	gitignorePath := snippetsDir + string(os.PathSeparator) + ".gitignore"
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		os.WriteFile(gitignorePath, []byte(".clidocs_git.json\n"), 0644)
	}

	if !isRepo {
		if _, err := run("init"); err != nil {
			return "", fmt.Errorf("git init failed")
		}
		if err := setIdentity(); err != nil {
			return "", err
		}
		run("remote", "add", "origin", cfg.RepoURL)
		run("branch", "-M", "main")

		// Fetch remote to check if it has content
		fetchOut, fetchErr := run("fetch", "origin")
		if fetchErr == nil {
			// Remote exists — check if it has a main branch
			remoteCheckOut, _ := run("ls-remote", "--heads", "origin", "main")
			if len(remoteCheckOut) > 0 {
				// Remote has content: pull it first (rebase so local changes stay on top)
				if _, err := run("pull", "--rebase", "origin", "main"); err != nil {
					// If rebase conflicts, abort and try merge strategy
					run("rebase", "--abort")
					if _, err2 := run("pull", "--allow-unrelated-histories", "origin", "main"); err2 != nil {
						return "", fmt.Errorf("could not merge remote content: %v", err2)
					}
				}
			}
		} else {
			_ = fetchOut
		}
	} else {
		if err := setIdentity(); err != nil {
			return "", err
		}
		run("remote", "set-url", "origin", cfg.RepoURL)

		// Always pull latest before pushing
		run("fetch", "origin")
		remoteCheckOut, _ := run("ls-remote", "--heads", "origin", "main")
		if len(remoteCheckOut) > 0 {
			if _, err := run("pull", "--rebase", "origin", "main"); err != nil {
				run("rebase", "--abort")
				run("pull", "--allow-unrelated-histories", "origin", "main")
			}
		}
	}

	// Stage everything
	if _, err := run("add", "-A"); err != nil {
		return "", fmt.Errorf("git add failed")
	}

	// Check if there's anything to commit
	statusOut, _ := run("status", "--porcelain")
	if statusOut == "" {
		return "Already up to date with remote. Nothing new to push.", nil
	}

	commitMsg := "clidocs: sync snippets"
	if out, err := run("commit", "-m", commitMsg); err != nil {
		return out, fmt.Errorf("git commit failed: %v", err)
	}

	out, err := run("push", "-u", "origin", "main")
	if err != nil {
		return out, fmt.Errorf("git push failed:\n%s", out)
	}

	return "Snippets synced to GitHub successfully!", nil
}
