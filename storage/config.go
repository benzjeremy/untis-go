package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Config represents the user configuration for the Untis application.
type Config struct {
	ActiveProfile     string `json:"activeProfile"`       // Profile ID (e.g. "1")
	Server            string `json:"server"`              // WebUntis Server e.g. "https://bk-technik-siegen.webuntis.com"
	School            string `json:"school"`              // School identifier e.g. "bk-technik-siegen"
	Username          string `json:"username"`            // Login username
	Password          string `json:"password,omitempty"`  // Login password (or fetched from keyring)
	AuthType          string `json:"authType"`            // "password" or "token"
	SelectedClassID   int    `json:"selectedClassId"`     // ID of currently active class
	SelectedClassName string `json:"selectedClassName"`   // Name of currently active class (e.g. "ITT125")
	Theme             string `json:"theme"`               // "system", "dark", "light"
	DefaultView       string `json:"defaultView"`         // "day", "week"
	Port              int    `json:"port"`                // Local web server port (default 8080)
}

// ConfigManager handles loading and persisting the configuration and timetable cache.
type ConfigManager struct {
	configDir string
	cacheDir  string
	filePath  string
	mu        sync.RWMutex
	current   Config
}

// NewConfigManager initializes and returns a ConfigManager pointing to ~/.config/untis-go
func NewConfigManager() (*ConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	configDir := filepath.Join(homeDir, ".config", "untis-go")
	cacheDir := filepath.Join(configDir, "cache")
	filePath := filepath.Join(configDir, "config.json")

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create config directory %s: %w", configDir, err)
	}

	cm := &ConfigManager{
		configDir: configDir,
		cacheDir:  cacheDir,
		filePath:  filePath,
		current: Config{
			ActiveProfile:     "1",
			Server:            "https://bk-technik-siegen.webuntis.com",
			School:            "bk-technik-siegen",
			Username:          "",
			Password:          "",
			AuthType:          "password",
			SelectedClassID:   0,
			SelectedClassName: "",
			Theme:             "dark",
			DefaultView:       "day",
			Port:              8080,
		},
	}

	// Try loading untis credentials from ~/.untis/data/credentials.json as baseline
	if cred, err := GetBestAvailableCredential(); err == nil && cred != nil {
		cm.current.ActiveProfile = cred.Profile
		cm.current.Server = cred.Server
		cm.current.School = cred.School
		cm.current.Username = cred.User
		cm.current.Password = cred.Password
		cm.current.AuthType = cred.Type
	}

	if err := cm.Load(); err != nil {
		// If untis-go config doesn't exist yet, save the initialized configuration
		_ = cm.Save(cm.current)
	}

	return cm, nil
}

// Load reads the configuration from disk.
func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		return err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("invalid config json: %w", err)
	}

	// If server/school missing or legacy demo values present, populate from credentials.json
	if cfg.Server == "" || cfg.School == "" || cfg.Server == "arche.webuntis.com" || cfg.School == "demo-schule" || cfg.Username == "" {
		if cred, err := GetBestAvailableCredential(); err == nil && cred != nil {
			cfg.Server = cred.Server
			cfg.School = cred.School
			cfg.Username = cred.User
			cfg.Password = cred.Password
			cfg.ActiveProfile = cred.Profile
			cfg.AuthType = cred.Type
			cfg.SelectedClassID = 0
			cfg.SelectedClassName = ""
		}
	}

	// If password in config is empty, attempt keyring lookup
	if cfg.Password == "" && cfg.Username != "" {
		cfg.Password = LookupKeyringPassword(cfg.Server, cfg.School, cfg.Username, cfg.AuthType)
	}

	// Sanity defaults
	if cfg.Port <= 0 {
		cfg.Port = 8080
	}
	if cfg.Theme == "" {
		cfg.Theme = "dark"
	}
	if cfg.DefaultView == "" {
		cfg.DefaultView = "day"
	}

	cm.current = cfg
	return nil
}

// Get returns a copy of current configuration.
func (cm *ConfigManager) Get() Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.current
}

// Save persists the provided configuration to disk.
func (cm *ConfigManager) Save(cfg Config) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cfg.Port <= 0 {
		cfg.Port = 8080
	}
	if cfg.Theme == "" {
		cfg.Theme = "dark"
	}
	if cfg.DefaultView == "" {
		cfg.DefaultView = "day"
	}

	cm.current = cfg

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cm.filePath, data, 0600)
}

// SaveCache saves arbitrary cache data as JSON
func (cm *ConfigManager) SaveCache(key string, data interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cleanKey := filepath.Clean(key)
	cacheFile := filepath.Join(cm.cacheDir, cleanKey+".json")

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, bytes, 0644)
}

// LoadCache loads cached JSON data if present
func (cm *ConfigManager) LoadCache(key string, target interface{}) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	cleanKey := filepath.Clean(key)
	cacheFile := filepath.Join(cm.cacheDir, cleanKey+".json")

	bytes, err := os.ReadFile(cacheFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(bytes, target)
}

// ClearCache removes all cached responses
func (cm *ConfigManager) ClearCache() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	_ = os.RemoveAll(cm.cacheDir)
	return os.MkdirAll(cm.cacheDir, 0755)
}

// GetConfigPath returns the absolute path to config.json
func (cm *ConfigManager) GetConfigPath() string {
	return cm.filePath
}
