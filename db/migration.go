package db

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LegacyCredentialsFile models ~/.untis/data/credentials.json
type LegacyCredentialsFile struct {
	Profiles map[string]struct {
		Name string `json:"name"`
	} `json:"profiles"`
	Credentials map[string]struct {
		User     string `json:"user"`
		Server   string `json:"server"`
		School   string `json:"school"`
		Type     string `json:"type"`
		Password string `json:"password,omitempty"`
	} `json:"credentials"`
	DefaultProfile string `json:"default-profile"`
}

// LegacyConfigFile models ~/.config/untis-go/config.json
type LegacyConfigFile struct {
	ActiveProfile     string `json:"activeProfile"`
	Server            string `json:"server"`
	School            string `json:"school"`
	Username          string `json:"username"`
	Password          string `json:"password,omitempty"`
	AuthType          string `json:"authType"`
	SelectedClassID   int    `json:"selectedClassId"`
	SelectedClassName string `json:"selectedClassName"`
	Theme             string `json:"theme"`
	DefaultView       string `json:"defaultView"`
	Port              int    `json:"port"`
}

// checkAndMigrate performs automated one-time migration from legacy JSON files
func (d *Database) checkAndMigrate() error {
	var profileCount int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM profiles`).Scan(&profileCount)
	if err != nil {
		return err
	}

	if profileCount > 0 {
		// Already migrated or populated
		return nil
	}

	log.Println("[DB Migration] Keine Profile in SQLite gefunden. Prüfe bestehende JSON-Konfigurationsdateien...")
	return d.MigrateFromLegacy()
}

// MigrateFromLegacy reads legacy JSON files and imports all profiles, settings, and cache into SQLite
func (d *Database) MigrateFromLegacy() error {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	legacyCredPath := filepath.Join(home, ".untis", "data", "credentials.json")
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		alt := filepath.Join(xdg, "credentials.json")
		if _, err := os.Stat(alt); err == nil {
			legacyCredPath = alt
		}
	}

	legacyConfigPath := filepath.Join(home, ".config", "untis-go", "config.json")

	// 1. Read legacy config.json if present
	var legacyCfg *LegacyConfigFile
	if cfgData, err := os.ReadFile(legacyConfigPath); err == nil {
		var cfg LegacyConfigFile
		if err := json.Unmarshal(cfgData, &cfg); err == nil {
			legacyCfg = &cfg
		}
	}

	// 2. Read legacy credentials.json if present
	var legacyCreds *LegacyCredentialsFile
	if credData, err := os.ReadFile(legacyCredPath); err == nil {
		var creds LegacyCredentialsFile
		if err := json.Unmarshal(credData, &creds); err == nil {
			legacyCreds = &creds
		}
	}

	migratedCount := 0

	// Determine active profile ID
	activeProfileID := "1"
	if legacyCfg != nil && legacyCfg.ActiveProfile != "" {
		activeProfileID = legacyCfg.ActiveProfile
	} else if legacyCreds != nil && legacyCreds.DefaultProfile != "" {
		activeProfileID = legacyCreds.DefaultProfile
	}

	// Migrate profiles from credentials.json
	if legacyCreds != nil && len(legacyCreds.Credentials) > 0 {
		for pid, cred := range legacyCreds.Credentials {
			pName := fmt.Sprintf("Profil %s", pid)
			if profInfo, ok := legacyCreds.Profiles[pid]; ok && profInfo.Name != "" {
				pName = profInfo.Name
			}

			pwd := cred.Password
			// If password missing in credentials.json, check legacy config.json
			if pwd == "" && legacyCfg != nil {
				if legacyCfg.ActiveProfile == pid || (legacyCfg.Username == cred.User && legacyCfg.School == cred.School) {
					pwd = legacyCfg.Password
				}
			}

			// If still missing, check secret-tool keyring
			if pwd == "" && cred.User != "" && cred.School != "" {
				pwd = lookupSecretToolPassword(cred.Server, cred.School, cred.User, cred.Type)
			}

			encPwd, err := EncryptPassword(pwd)
			if err != nil {
				log.Printf("[DB Migration] Warnung: Passwort für Profil %s konnte nicht verschlüsselt werden: %v", pid, err)
			}

			isActive := pid == activeProfileID
			prof := Profile{
				ID:                pid,
				Name:              pName,
				School:            cred.School,
				Server:            cred.Server,
				Username:          cred.User,
				EncryptedPassword: encPwd,
				IsActive:          isActive,
			}

			if err := d.SaveProfile(&prof); err != nil {
				log.Printf("[DB Migration] Fehler beim Importieren von Profil %s: %v", pid, err)
			} else {
				migratedCount++
			}
		}
	} else if legacyCfg != nil && legacyCfg.Username != "" {
		// If credentials.json was not found but config.json has account details
		encPwd, _ := EncryptPassword(legacyCfg.Password)
		pID := legacyCfg.ActiveProfile
		if pID == "" {
			pID = "1"
		}
		prof := Profile{
			ID:                pID,
			Name:              fmt.Sprintf("Profil %s (%s)", pID, legacyCfg.School),
			School:            legacyCfg.School,
			Server:            legacyCfg.Server,
			Username:          legacyCfg.Username,
			EncryptedPassword: encPwd,
			IsActive:          true,
		}
		_ = d.SaveProfile(&prof)
		migratedCount++
	}

	// Migrate settings
	if legacyCfg != nil {
		if legacyCfg.Theme != "" {
			_ = d.SetSetting("theme", legacyCfg.Theme)
		} else {
			_ = d.SetSetting("theme", "dark")
		}

		if legacyCfg.DefaultView != "" {
			_ = d.SetSetting("default_view", legacyCfg.DefaultView)
		} else {
			_ = d.SetSetting("default_view", "day")
		}

		if legacyCfg.SelectedClassID > 0 {
			_ = d.SetIntSetting("selected_class_id", legacyCfg.SelectedClassID)
		}
		if legacyCfg.SelectedClassName != "" {
			_ = d.SetSetting("selected_class_name", legacyCfg.SelectedClassName)
		}
		if legacyCfg.Port > 0 {
			_ = d.SetIntSetting("port", legacyCfg.Port)
		} else {
			_ = d.SetIntSetting("port", 8080)
		}
	} else {
		_ = d.SetSetting("theme", "dark")
		_ = d.SetSetting("default_view", "day")
		_ = d.SetIntSetting("port", 8080)
	}

	_ = d.SetSetting("active_profile", activeProfileID)

	// Migrate cached classes from ~/.config/untis-go/cache/classes_*.json
	legacyCacheDir := filepath.Join(home, ".config", "untis-go", "cache")
	if files, err := os.ReadDir(legacyCacheDir); err == nil {
		for _, file := range files {
			if strings.HasPrefix(file.Name(), "classes_") && strings.HasSuffix(file.Name(), ".json") {
				school := strings.TrimSuffix(strings.TrimPrefix(file.Name(), "classes_"), ".json")
				data, err := os.ReadFile(filepath.Join(legacyCacheDir, file.Name()))
				if err == nil {
					var rawClasses []struct {
						ID       int    `json:"id"`
						Name     string `json:"name"`
						LongName string `json:"longName"`
						Active   bool   `json:"active"`
					}
					if err := json.Unmarshal(data, &rawClasses); err == nil && len(rawClasses) > 0 {
						var classes []Class
						for _, rc := range rawClasses {
							classes = append(classes, Class{
								ID:       rc.ID,
								School:   school,
								Name:     rc.Name,
								LongName: rc.LongName,
								Active:   rc.Active,
							})
						}
						_ = d.SaveClasses(school, classes)
						log.Printf("[DB Migration] %d Klassen für Schule '%s' importiert.", len(classes), school)
					}
				}
			}

			// Migrate cached timetable files (tt_*.json)
			if strings.HasPrefix(file.Name(), "tt_") && strings.HasSuffix(file.Name(), ".json") {
				// Format: tt_<school>_<classID>_<startDate>_<endDate>.json
				parts := strings.Split(strings.TrimSuffix(file.Name(), ".json"), "_")
				if len(parts) >= 4 {
					var classID int
					fmt.Sscanf(parts[2], "%d", &classID)
					if classID > 0 {
						data, err := os.ReadFile(filepath.Join(legacyCacheDir, file.Name()))
						if err == nil {
							// Each file contains an array of lessons with "date": "YYYY-MM-DD"
							var lessons []map[string]interface{}
							if err := json.Unmarshal(data, &lessons); err == nil && len(lessons) > 0 {
								// Group by date
								byDate := make(map[string][]map[string]interface{})
								for _, l := range lessons {
									if dStr, ok := l["date"].(string); ok && dStr != "" {
										byDate[dStr] = append(byDate[dStr], l)
									}
								}
								for dStr, dLessons := range byDate {
									if dJSON, err := json.Marshal(dLessons); err == nil {
										_ = d.SaveTimetableCache(classID, dStr, string(dJSON))
									}
								}
							}
						}
					}
				}
			}
		}
	}

	log.Printf("[DB Migration] Migration erfolgreich abgeschlossen! %d Profile in SQLite überführt.", migratedCount)
	return nil
}

// lookupSecretToolPassword searches libsecret for stored untis password
func lookupSecretToolPassword(server, school, user, credType string) string {
	if credType == "" {
		credType = "password"
	}
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
