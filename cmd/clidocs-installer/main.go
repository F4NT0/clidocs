package main

import (
	"fmt"
	"os"

	"clidocs/internal/installer"
)

const logo = `
 ██████╗██╗     ██╗██████╗  ██████╗  ██████╗███████╗
██╔════╝██║     ██║██╔══██╗██╔═══██╗██╔════╝██╔════╝
██║     ██║     ██║██║  ██║██║   ██║██║     ███████╗
██║     ██║     ██║██║  ██║██║   ██║██║     ╚════██║
╚██████╗███████╗██║██████╔╝╚██████╔╝╚██████╗███████║
 ╚═════╝╚══════╝╚═╝╚═════╝  ╚═════╝  ╚═════╝╚══════╝`

func main() {
	skip := len(os.Args) > 1 && (os.Args[1] == "-y" || os.Args[1] == "--yes")
	if err := installer.Run(installer.Config{Logo: logo, Version: "(latest)"}, skip); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
