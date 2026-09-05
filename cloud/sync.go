package cloud

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/benzjeremy/untis-go/db"
	"golang.org/x/crypto/pbkdf2"
)

var cloudSalt = []byte("untis-go-onedrive-cloud-sync-salt-v1.6.0-secure")

// CloudProfile stores a profile in OneDrive with AES-256-GCM encrypted password (never in plaintext)
type CloudProfile struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	School            string `json:"school"`
	Server            string `json:"server"`
	Username          string `json:"username"`
	EncryptedPassword string `json:"encryptedPassword,omitempty"` // AES-256-GCM ciphertext
	IsActive          bool   `json:"isActive"`
	CreatedAt         string `json:"createdAt"`
}

// ConfigPayload encapsulates all user configurations stored in SQLite for OneDrive backup
type ConfigPayload struct {
	Version      string            `json:"version"`
	ExportedAt   string            `json:"exportedAt"`
	AccountEmail string            `json:"accountEmail"`
	Settings     map[string]string `json:"settings"`
	Profiles     []CloudProfile    `json:"profiles"`
	Homework     []db.Homework     `json:"homework"`
	Absences     []db.Absence      `json:"absences"`
	Aliases      map[string]string `json:"aliases"` // key: "profileID:subject" -> alias
}

func deriveCloudKey(email, userID string) []byte {
	var data []byte
	data = append(data, cloudSalt...)
	data = append(data, []byte(strings.ToLower(strings.TrimSpace(email)))...)
	data = append(data, []byte(strings.TrimSpace(userID))...)
	if len(data) == len(cloudSalt) {
		data = append(data, []byte("untis-go-default-cloud-secret-key")...)
	}
	return pbkdf2.Key(data, cloudSalt, 100000, 32, sha256.New)
}

func encryptCloudPassword(plaintext, email, userID string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key := deriveCloudKey(email, userID)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func decryptCloudPassword(ciphertext, email, userID string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	key := deriveCloudKey(email, userID)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ungültige ciphertext-länge")
	}
	nonce, cipherBytes := data[:nonceSize], data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, cipherBytes, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// ExportLocalConfig bundles local settings, profiles (with encrypted credentials), homework, absences, and aliases into a JSON payload
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

	msEmail := cleanSettings["ms_user_email"]
	msUserID := cleanSettings["ms_user_id"]

	var cloudProfiles []CloudProfile
	for _, p := range profiles {
		encPass, errEnc := encryptCloudPassword(p.Password, msEmail, msUserID)
		if errEnc != nil {
			encPass = p.EncryptedPassword
		}
		cloudProfiles = append(cloudProfiles, CloudProfile{
			ID:                p.ID,
			Name:              p.Name,
			School:            p.School,
			Server:            p.Server,
			Username:          p.Username,
			EncryptedPassword: encPass,
			IsActive:          p.IsActive,
			CreatedAt:         p.CreatedAt.Format(time.RFC3339),
		})
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
		Profiles:     cloudProfiles,
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

	msEmail := payload.Settings["ms_user_email"]
	msUserID := payload.Settings["ms_user_id"]

	// 2. Restore Profiles with decrypted credentials re-encrypted for local hardware
	for _, cp := range payload.Profiles {
		var plainPass string
		if cp.EncryptedPassword != "" {
			if dec, err := decryptCloudPassword(cp.EncryptedPassword, msEmail, msUserID); err == nil {
				plainPass = dec
			}
		}

		p := db.Profile{
			ID:       cp.ID,
			Name:     cp.Name,
			School:   cp.School,
			Server:   cp.Server,
			Username: cp.Username,
			Password: plainPass,
			IsActive: cp.IsActive,
		}
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
