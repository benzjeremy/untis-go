package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// UntisProfile represents a user profile name in ~/.untis/data/credentials.json
type UntisProfile struct {
	Name string `json:"name"`
}

// UntisCredential holds the login details for a WebUntis profile
type UntisCredential struct {
	User     string `json:"user"`
	Server   string `json:"server"`
	School   string `json:"school"`
	Type     string `json:"type"`
	Password string `json:"password,omitempty"`
	Profile  string `json:"profile,omitempty"`
}

// UntisDataFile models the ~/.untis/data/credentials.json file structure
type UntisDataFile struct {
	Profiles       map[string]UntisProfile    `json:"profiles"`
	Credentials    map[string]UntisCredential `json:"credentials"`
	DefaultProfile string                     `json:"default-profile"`
}

// GetUntisCredentialsPath returns the path to ~/.untis/data/credentials.json
func GetUntisCredentialsPath() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "credentials.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".untis", "data", "credentials.json")
}

// LoadUntisData reads and parses ~/.untis/data/credentials.json
func LoadUntisData() (*UntisDataFile, error) {
	path := GetUntisCredentialsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read untis credentials file: %w", err)
	}

	var df UntisDataFile
	if err := json.Unmarshal(data, &df); err != nil {
		return nil, fmt.Errorf("could not parse untis credentials file: %w", err)
	}

	if df.Profiles == nil {
		df.Profiles = make(map[string]UntisProfile)
	}
	if df.Credentials == nil {
		df.Credentials = make(map[string]UntisCredential)
	}

	return &df, nil
}

// SaveUntisData persists the UntisDataFile back to disk
func SaveUntisData(df *UntisDataFile) error {
	path := GetUntisCredentialsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(df, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// LookupKeyringPassword retrieves the password from Secret Service using secret-tool
func LookupKeyringPassword(server, school, user, credType string) string {
	// Try exact server
	servers := []string{server}
	if strings.HasPrefix(server, "https://") {
		servers = append(servers, strings.TrimPrefix(server, "https://"))
	} else {
		servers = append(servers, "https://"+server)
	}

	for _, s := range servers {
		cmd := exec.Command("secret-tool", "lookup", "server", s, "school", school, "user", user, "type", credType)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			pwd := strings.TrimSpace(out.String())
			if pwd != "" {
				return pwd
			}
		}
	}
	return ""
}

// StoreKeyringPassword stores the password into Secret Service using secret-tool
func StoreKeyringPassword(server, school, user, credType, password string) error {
	normServer := server
	if !strings.HasPrefix(normServer, "https://") {
		normServer = "https://" + normServer
	}

	cmd := exec.Command("secret-tool", "store",
		"--label=Untis Password",
		"server", normServer,
		"school", school,
		"user", user,
		"type", credType,
	)
	cmd.Stdin = strings.NewReader(password)
	return cmd.Run()
}

// GetResolvedCredential returns the credential for profileID with password filled in
func GetResolvedCredential(profileID string) (*UntisCredential, error) {
	df, err := LoadUntisData()
	if err != nil {
		return nil, err
	}

	cred, ok := df.Credentials[profileID]
	if !ok {
		return nil, fmt.Errorf("profile %s not found", profileID)
	}

	cred.Profile = profileID
	if cred.Type == "" {
		cred.Type = "password"
	}

	if cred.Password == "" {
		// Lookup from secret-tool keyring
		cred.Password = LookupKeyringPassword(cred.Server, cred.School, cred.User, cred.Type)
	}

	if !strings.HasPrefix(cred.Server, "https://") && !strings.HasPrefix(cred.Server, "http://") {
		cred.Server = "https://" + cred.Server
	}

	return &cred, nil
}

// GetBestAvailableCredential returns a usable credential (preferring profile with password)
func GetBestAvailableCredential() (*UntisCredential, error) {
	df, err := LoadUntisData()
	if err != nil {
		return nil, err
	}

	// 1. Try default profile if it has a password
	if df.DefaultProfile != "" {
		if cred, err := GetResolvedCredential(df.DefaultProfile); err == nil && cred.Password != "" {
			return cred, nil
		}
	}

	// 2. Try any profile that has a valid password
	for pid := range df.Credentials {
		if cred, err := GetResolvedCredential(pid); err == nil && cred.Password != "" {
			return cred, nil
		}
	}

	// 3. Fallback to default profile even if password is empty (e.g. anonymous)
	if df.DefaultProfile != "" {
		return GetResolvedCredential(df.DefaultProfile)
	}

	// 4. Fallback to first profile
	for pid := range df.Credentials {
		return GetResolvedCredential(pid)
	}

	return nil, fmt.Errorf("no untis profiles found")
}
