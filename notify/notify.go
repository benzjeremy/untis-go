package notify

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"

	"github.com/benzjeremy/untis-go/diff"
)

// SendNotification sends a native operating system notification
func SendNotification(title, message string) error {
	switch runtime.GOOS {
	case "linux":
		return sendLinux(title, message)
	case "windows":
		return sendWindows(title, message)
	case "darwin":
		return sendDarwin(title, message)
	default:
		log.Printf("[Notification] %s: %s\n", title, message)
		return nil
	}
}

// NotifyLessonChange sends an OS notification for a detected lesson change
func NotifyLessonChange(change diff.LessonChange) error {
	return SendNotification(change.Title, change.Message)
}

func sendLinux(title, message string) error {
	if path, err := exec.LookPath("notify-send"); err == nil {
		cmd := exec.Command(path, "-a", "untis-go", "-i", "untis-go", "-u", "normal", title, message)
		if err := cmd.Run(); err != nil {
			log.Printf("[Notify] notify-send error: %v\n", err)
			return err
		}
		return nil
	}
	log.Printf("[Notify/Fallback] %s: %s\n", title, message)
	return nil
}

func sendWindows(title, message string) error {
	// PowerShell balloon / toast notification
	escapedTitle := strings.ReplaceAll(title, "'", "''")
	escapedMsg := strings.ReplaceAll(message, "'", "''")
	script := fmt.Sprintf(`[reflection.assembly]::loadwithpartialname('System.Windows.Forms') | Out-Null; $notify = New-Object System.Windows.Forms.NotifyIcon; $notify.Icon = [System.Drawing.SystemIcons]::Information; $notify.BalloonTipTitle = '%s'; $notify.BalloonTipText = '%s'; $notify.Visible = $True; $notify.ShowBalloonTip(5000); Start-Sleep -Seconds 1; $notify.Dispose()`, escapedTitle, escapedMsg)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	if err := cmd.Start(); err != nil {
		log.Printf("[Notify/Windows] powershell error: %v\n", err)
		return err
	}
	return nil
}

func sendDarwin(title, message string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, strings.ReplaceAll(message, `"`, `\"`), strings.ReplaceAll(title, `"`, `\"`))
	return exec.Command("osascript", "-e", script).Run()
}
