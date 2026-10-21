# clidocs - Terminal-Native Snippet Manager

clidocs is a keyboard-driven terminal application built with Go that helps developers organize and manage code snippets efficiently. 

## What it does

clidocs provides a three-panel interface (Folders / Snippets / Preview) that allows you to:
- **Organize snippets** into folders and subfolders with easy navigation
- **Create, edit, and search** code files with syntax highlighting
- **Preview** code snippets with GitHub Dark theme syntax highlighting
- **Open files** directly in Neovim or VS Code for editing
- **Import files** from anywhere on your system
- **Sync snippets** to GitHub repositories
- **Search and filter** folders and files in real-time

## Who it's for

Developers who prefer working in the terminal and want a fast, keyboard-centric way to manage reusable code snippets, documentation, and reference materials without leaving their command-line environment.

## Platform

Built for Windows 11 with PowerShell, requires Go 1.24+ for building from source. Snippets are stored locally in `%USERPROFILE%\clidocs_snippets\`.
