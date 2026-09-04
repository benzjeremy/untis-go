//go:build (!linux && !windows) || (linux && !cgo)

package main

import (
	"log"
	"os/exec"
)

func LaunchGUI(title, url string, width, height int, forceBrowser bool) {
	OpenBrowser(url)
}

func OpenBrowser(url string) {
	_ = exec.Command("xdg-open", url).Start()
	log.Printf("[Browser] URL geöffnet: %s\n", url)
}
