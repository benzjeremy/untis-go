//go:build windows

package main

import (
	"log"
	"os/exec"
)

// LaunchGUI opens the system browser in app mode on Windows
func LaunchGUI(title, url string, width, height int, forceBrowser bool) {
	log.Println("[GUI] Starte Untis Desktop im Anwendungsmodus unter Windows...")
	OpenBrowser(url)
}

// OpenBrowser opens the URL in Edge or Chrome in app-window mode, falling back to default browser
func OpenBrowser(url string) {
	commands := [][]string{
		{"msedge.exe", "--app=" + url},
		{"chrome.exe", "--app=" + url},
		{"cmd.exe", "/c", "start", url},
	}

	for _, cmdArgs := range commands {
		if path, err := exec.LookPath(cmdArgs[0]); err == nil {
			cmd := exec.Command(path, cmdArgs[1:]...)
			if err := cmd.Start(); err == nil {
				log.Printf("[Browser] Geöffnet mit %s (%s)\n", cmdArgs[0], url)
				return
			}
		}
	}

	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	log.Printf("[Browser] URL im Standardbrowser geöffnet: %s\n", url)
}
