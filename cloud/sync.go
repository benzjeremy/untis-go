package cloud

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/benzjeremy/untis-go/db"
)

// ConfigPayload encapsulates all user configurations stored in SQLite for OneDrive backup
type ConfigPayload struct {
	Version      string            `json:"version"`
	ExportedAt   string            `json:"exportedAt"`
	AccountEmail string            `json:"accountEmail"`
	Settings     map[string]string `json:"settings"`
	Profiles     []db.Profile      `json:"profiles"`
	Homework     []db.Homework     `json:"homework"`
	Absences     []db.Absence      `json:"absences"`
	Aliases      map[string]string `json:"aliases"` // key: "profileID:subject" -> alias
}

// ExportLocalConfig bundles local settings, profiles, homework, absences, and aliases into a JSON payload
func ExportLocalConfig(database *db.Database) ([]byte, error) {
	if database == nil {
		return nil, fmt.Errorf("datenbank nicht verfügbar")
	}

	settings, err := database.GetAllSettings()
	if err != nil {
		return nil, fmt.Errorf("fehler beim laden der einstellungen: %w", err)
	}

	// Filter out ephemeral tokens from the exported settings
	cleanSettings := make(map[string]string)
	for k, v := range settings {
		if k == "ms_access_token" || k == "ms_refresh_token" || k == "ms_token_expiry" {
			continue
		}
		cleanSettings[k] = v
	}

	profiles, err := database.GetProfiles()
	if err != nil {
		return nil, fmt.Errorf("fehler beim laden der profile: %w", err)
	}

	var allHomework []db.Homework
	var allAbsences []db.Absence
	aliasesMap := make(map[string]string)

	for _, p := range profiles {
		if hw, errHw := database.GetHomeworks(p.ID); errHw == nil {
			allHomework = append(allHomework, hw...)
		}
		if abs, errAbs := database.GetAbsences(p.ID); errAbs == nil {
			allAbsences = append(allAbsences, abs...)
		}
		if al, errAl := database.GetSubjectAliases(p.ID); errAl == nil {
			for subj, alias := range al {
				aliasesMap[p.ID+":"+subj] = alias
			}
		}
	}

	payload := ConfigPayload{
		Version:      "1.6.0",
		ExportedAt:   time.Now().UTC().Format(time.RFC3339),
		AccountEmail: cleanSettings["ms_user_email"],
		Settings:     cleanSettings,
		Profiles:     profiles,
		Homework:     allHomework,
		Absences:     allAbsences,
		Aliases:      aliasesMap,
	}

	return json.MarshalIndent(payload, "", "  ")
}

// ImportConfigToLocal restores a ConfigPayload into the local SQLite database
func ImportConfigToLocal(database *db.Database, data []byte) error {
	if database == nil {
		return fmt.Errorf("datenbank nicht verfügbar")
	}

	var payload ConfigPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("ungültiges konfigurationsformat: %w", err)
	}

	// 1. Restore Settings (without overwriting current active ms tokens)
	for k, v := range payload.Settings {
		if k == "ms_access_token" || k == "ms_refresh_token" || k == "ms_token_expiry" || k == "ms_logged_in" {
			continue
		}
		_ = database.SetSetting(k, v)
	}

	// 2. Restore Profiles
	for _, p := range payload.Profiles {
		_ = database.SaveProfile(&p)
	}

	// 3. Restore Homework
	for _, hw := range payload.Homework {
		_ = database.CreateHomework(&hw)
	}

	// 4. Restore Absences
	for _, abs := range payload.Absences {
		_ = database.CreateAbsence(&abs)
	}

	// 5. Restore Aliases
	for key, alias := range payload.Aliases {
		var profID, subj string
		n, _ := fmt.Sscanf(key, "%s:%s", &profID, &subj)
		if n == 2 && profID != "" && subj != "" {
			_ = database.SetSubjectAlias(profID, subj, alias)
		}
	}

	return nil
}
