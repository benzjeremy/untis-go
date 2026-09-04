package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ReleaseInfo holds details about available updates from GitHub
type ReleaseInfo struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	HasUpdate      bool   `json:"hasUpdate"`
	Title          string `json:"title"`
	ReleaseNotes   string `json:"releaseNotes"`
	ReleaseURL     string `json:"releaseUrl"`
	DownloadURL    string `json:"downloadUrl"`
	PublishedAt    string `json:"publishedAt"`
	IsPrerelease   bool   `json:"isPrerelease"`
}

type gitHubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Prerelease  bool   `json:"prerelease"`
	Draft       bool   `json:"draft"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// CheckForUpdate queries GitHub for the latest release (including pre-releases)
func CheckForUpdate(currentVersion string) (*ReleaseInfo, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/repos/benzjeremy/untis-go/releases", nil)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Erstellen der Update-Anfrage: %w", err)
	}
	req.Header.Set("User-Agent", "untis-go-updater/"+currentVersion)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github api nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api antwortete mit status %d", resp.StatusCode)
	}

	var releases []gitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("fehler beim parsen der release-daten: %w", err)
	}

	if len(releases) == 0 {
		return &ReleaseInfo{
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
			HasUpdate:      false,
		}, nil
	}

	// Pick newest published non-draft release
	var targetRelease *gitHubRelease
	for i := range releases {
		if !releases[i].Draft {
			targetRelease = &releases[i]
			break
		}
	}

	if targetRelease == nil {
		return &ReleaseInfo{
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
			HasUpdate:      false,
		}, nil
	}

	cleanLatest := strings.TrimPrefix(targetRelease.TagName, "v")
	cleanCurrent := strings.TrimPrefix(currentVersion, "v")

	hasUpdate := compareVersions(cleanLatest, cleanCurrent) > 0

	// Find suitable download URL for current OS & ARCH
	var downloadURL string
	targetOS := runtime.GOOS
	targetArch := runtime.GOARCH

	for _, a := range targetRelease.Assets {
		name := strings.ToLower(a.Name)
		if strings.Contains(name, targetOS) {
			if targetArch == "amd64" && (strings.Contains(name, "amd64") || strings.Contains(name, "x86_64")) {
				downloadURL = a.BrowserDownloadURL
				break
			}
			if targetArch == "arm64" && (strings.Contains(name, "arm64") || strings.Contains(name, "aarch64")) {
				downloadURL = a.BrowserDownloadURL
				break
			}
		}
	}

	// Fallback if OS matched without explicit arch
	if downloadURL == "" {
		for _, a := range targetRelease.Assets {
			name := strings.ToLower(a.Name)
			if strings.Contains(name, targetOS) {
				downloadURL = a.BrowserDownloadURL
				break
			}
		}
	}

	return &ReleaseInfo{
		CurrentVersion: cleanCurrent,
		LatestVersion:  cleanLatest,
		HasUpdate:      hasUpdate,
		Title:          targetRelease.Name,
		ReleaseNotes:   targetRelease.Body,
		ReleaseURL:     targetRelease.HTMLURL,
		DownloadURL:    downloadURL,
		PublishedAt:    targetRelease.PublishedAt,
		IsPrerelease:   targetRelease.Prerelease,
	}, nil
}

// ApplyUpdate downloads and installs the update into the current executable
func ApplyUpdate(downloadURL string) error {
	if downloadURL == "" {
		return fmt.Errorf("keine download-url für diese plattform verfügbar")
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("ausführbare datei konnte nicht ermittelt werden: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("symlink der ausführbaren datei konnte nicht aufgelöst werden: %w", err)
	}

	// Download archive
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("anfrage konnte nicht erstellt werden: %w", err)
	}
	req.Header.Set("User-Agent", "untis-go-updater")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download fehlgeschlagen mit status: %d", resp.StatusCode)
	}

	archiveData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("download-daten konnten nicht gelesen werden: %w", err)
	}

	// Extract binary
	var newBinary []byte
	lowerURL := strings.ToLower(downloadURL)

	if strings.HasSuffix(lowerURL, ".tar.gz") || strings.HasSuffix(lowerURL, ".tgz") {
		newBinary, err = extractFromTarGz(archiveData)
	} else if strings.HasSuffix(lowerURL, ".zip") {
		newBinary, err = extractFromZip(archiveData)
	} else {
		// Assume raw binary
		newBinary = archiveData
	}

	if err != nil {
		return fmt.Errorf("archiv konnte nicht entpackt werden: %w", err)
	}

	if len(newBinary) == 0 {
		return fmt.Errorf("keine gültige untis-go binary im archiv gefunden")
	}

	// Replace executable
	return replaceExecutable(execPath, newBinary)
}

func extractFromTarGz(data []byte) ([]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		baseName := filepath.Base(header.Name)
		if baseName == "untis-go" || baseName == "untis-go.exe" {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("binary untis-go nicht im .tar.gz gefunden")
}

func extractFromZip(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	for _, f := range zr.File {
		baseName := filepath.Base(f.Name)
		if baseName == "untis-go" || baseName == "untis-go.exe" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("binary untis-go nicht in der .zip datei gefunden")
}

func replaceExecutable(targetPath string, newContent []byte) error {
	dir := filepath.Dir(targetPath)

	// Write to temporary file in the same directory (guarantees same filesystem)
	tempFile, err := os.CreateTemp(dir, "untis-update-*")
	if err != nil {
		return fmt.Errorf("temp-datei in %s konnte nicht erstellt werden (fehlende schreibrechte?): %w", dir, err)
	}
	tempPath := tempFile.Name()

	if _, err := tempFile.Write(newContent); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("neue binary konnte nicht geschrieben werden: %w", err)
	}
	_ = tempFile.Close()

	// Ensure executable permissions
	_ = os.Chmod(tempPath, 0755)

	// On Windows, running files cannot be overwritten, but CAN be renamed to .old
	if runtime.GOOS == "windows" {
		oldPath := targetPath + ".old"
		_ = os.Remove(oldPath) // remove existing .old if present
		if err := os.Rename(targetPath, oldPath); err != nil {
			_ = os.Remove(tempPath)
			return fmt.Errorf("bestehende exe konnte nicht umbenannt werden: %w", err)
		}
		if err := os.Rename(tempPath, targetPath); err != nil {
			// Rollback
			_ = os.Rename(oldPath, targetPath)
			_ = os.Remove(tempPath)
			return fmt.Errorf("neue exe konnte nicht an zielort verschoben werden: %w", err)
		}
		return nil
	}

	// On Linux / macOS, atomic rename
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("aktualisierte datei konnte nicht platziert werden: %w", err)
	}

	return nil
}

// CompareVersions compares two semver strings like "1.1.0" and "1.0.0".
// Returns 1 if a > b, -1 if a < b, 0 if a == b.
func CompareVersions(a, b string) int {
	return compareVersions(a, b)
}

func compareVersions(a, b string) int {
	aParts := parseVersionParts(a)
	bParts := parseVersionParts(b)

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		aVal := 0
		if i < len(aParts) {
			aVal = aParts[i]
		}
		bVal := 0
		if i < len(bParts) {
			bVal = bParts[i]
		}
		if aVal > bVal {
			return 1
		}
		if aVal < bVal {
			return -1
		}
	}
	return 0
}

func parseVersionParts(v string) []int {
	v = strings.TrimPrefix(v, "v")
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}
	segments := strings.Split(v, ".")
	var parts []int
	for _, seg := range segments {
		num, _ := strconv.Atoi(seg)
		parts = append(parts, num)
	}
	return parts
}
