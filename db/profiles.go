package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// GetProfiles retrieves all profiles with decrypted passwords in memory
func (d *Database) GetProfiles() ([]Profile, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, name, school, server, username, encrypted_password, is_active, created_at 
		FROM profiles 
		ORDER BY is_active DESC, created_at ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("error querying profiles: %w", err)
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var p Profile
		var isActiveInt int
		var createdAtStr string

		if err := rows.Scan(&p.ID, &p.Name, &p.School, &p.Server, &p.Username, &p.EncryptedPassword, &isActiveInt, &createdAtStr); err != nil {
			return nil, fmt.Errorf("error scanning profile: %w", err)
		}

		p.IsActive = isActiveInt == 1
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if p.CreatedAt.IsZero() {
			p.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		}

		// Decrypt password into memory
		if p.EncryptedPassword != "" {
			if decrypted, err := DecryptPassword(p.EncryptedPassword); err == nil {
				p.Password = decrypted
			}
		}

		profiles = append(profiles, p)
	}

	return profiles, nil
}

// GetProfile retrieves a single profile by ID
func (d *Database) GetProfile(id string) (*Profile, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	row := d.db.QueryRow(`
		SELECT id, name, school, server, username, encrypted_password, is_active, created_at 
		FROM profiles 
		WHERE id = ?
	`, id)

	var p Profile
	var isActiveInt int
	var createdAtStr string

	if err := row.Scan(&p.ID, &p.Name, &p.School, &p.Server, &p.Username, &p.EncryptedPassword, &isActiveInt, &createdAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("profile with id %s not found", id)
		}
		return nil, err
	}

	p.IsActive = isActiveInt == 1
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
	if p.CreatedAt.IsZero() {
		p.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	}

	if p.EncryptedPassword != "" {
		if decrypted, err := DecryptPassword(p.EncryptedPassword); err == nil {
			p.Password = decrypted
		}
	}

	return &p, nil
}

// GetActiveProfile returns the currently active user profile
func (d *Database) GetActiveProfile() (*Profile, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// First try profile with is_active = 1
	row := d.db.QueryRow(`
		SELECT id, name, school, server, username, encrypted_password, is_active, created_at 
		FROM profiles 
		WHERE is_active = 1 
		LIMIT 1
	`)

	var p Profile
	var isActiveInt int
	var createdAtStr string

	err := row.Scan(&p.ID, &p.Name, &p.School, &p.Server, &p.Username, &p.EncryptedPassword, &isActiveInt, &createdAtStr)
	if err == nil {
		p.IsActive = true
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if p.CreatedAt.IsZero() {
			p.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		}
		if p.EncryptedPassword != "" {
			if decrypted, err := DecryptPassword(p.EncryptedPassword); err == nil {
				p.Password = decrypted
			}
		}
		return &p, nil
	}

	// Fallback to first profile if none is marked active
	rowFallback := d.db.QueryRow(`
		SELECT id, name, school, server, username, encrypted_password, is_active, created_at 
		FROM profiles 
		ORDER BY id ASC 
		LIMIT 1
	`)

	if err := rowFallback.Scan(&p.ID, &p.Name, &p.School, &p.Server, &p.Username, &p.EncryptedPassword, &isActiveInt, &createdAtStr); err != nil {
		return nil, fmt.Errorf("no profiles found: %w", err)
	}

	p.IsActive = isActiveInt == 1
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
	if p.EncryptedPassword != "" {
		if decrypted, err := DecryptPassword(p.EncryptedPassword); err == nil {
			p.Password = decrypted
		}
	}

	return &p, nil
}

// SetActiveProfile sets the specified profile as active and marks all others inactive
func (d *Database) SetActiveProfile(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE profiles SET is_active = 0`); err != nil {
		return err
	}

	res, err := tx.Exec(`UPDATE profiles SET is_active = 1 WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		return fmt.Errorf("profile %s does not exist", id)
	}

	// Also record active profile ID in settings table
	_, _ = tx.Exec(`INSERT INTO settings(key, value) VALUES ('active_profile', ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, id)

	return tx.Commit()
}

// SaveProfile inserts or updates a profile, automatically encrypting the password
func (d *Database) SaveProfile(p *Profile) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Clean inputs
	cleanServer := strings.TrimSpace(p.Server)
	if cleanServer != "" && !strings.HasPrefix(cleanServer, "http://") && !strings.HasPrefix(cleanServer, "https://") {
		cleanServer = "https://" + cleanServer
	}
	cleanServer = strings.TrimSuffix(cleanServer, "/")
	p.Server = cleanServer
	p.School = strings.TrimSpace(p.School)
	p.Username = strings.TrimSpace(p.Username)

	// If password provided in plain text, encrypt it
	if p.Password != "" {
		enc, err := EncryptPassword(p.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}
		p.EncryptedPassword = enc
	}

	isActiveInt := 0
	if p.IsActive {
		isActiveInt = 1
	}

	query := `
		INSERT INTO profiles (id, name, school, server, username, encrypted_password, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			school = excluded.school,
			server = excluded.server,
			username = excluded.username,
			encrypted_password = CASE WHEN excluded.encrypted_password != '' THEN excluded.encrypted_password ELSE profiles.encrypted_password END,
			is_active = excluded.is_active
	`

	_, err := d.db.Exec(query, p.ID, p.Name, p.School, p.Server, p.Username, p.EncryptedPassword, isActiveInt)
	return err
}

// DeleteProfile removes a profile from SQLite and cascades to local homework/absences
func (d *Database) DeleteProfile(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if this was the active profile
	var isActive int
	_ = d.db.QueryRow(`SELECT is_active FROM profiles WHERE id = ?`, id).Scan(&isActive)

	// Clean up related data
	_, _ = d.db.Exec(`DELETE FROM homework WHERE profile_id = ?`, id)
	_, _ = d.db.Exec(`DELETE FROM absences WHERE profile_id = ?`, id)

	// Delete profile
	_, err := d.db.Exec(`DELETE FROM profiles WHERE id = ?`, id)
	if err != nil {
		return err
	}

	// If this was the active profile, promote another profile if one exists
	if isActive == 1 {
		var nextID string
		err := d.db.QueryRow(`SELECT id FROM profiles ORDER BY created_at ASC LIMIT 1`).Scan(&nextID)
		if err == nil && nextID != "" {
			_, _ = d.db.Exec(`UPDATE profiles SET is_active = 1 WHERE id = ?`, nextID)
			_, _ = d.db.Exec(`INSERT INTO settings(key, value) VALUES ('active_profile', ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, nextID)
		} else {
			_, _ = d.db.Exec(`DELETE FROM settings WHERE key = 'active_profile'`)
		}
	}

	return nil
}

// GetDecryptedPassword returns the decrypted password for a profile
func (d *Database) GetDecryptedPassword(p *Profile) (string, error) {
	if p == nil {
		return "", nil
	}
	if p.Password != "" {
		return p.Password, nil
	}
	return DecryptPassword(p.EncryptedPassword)
}

