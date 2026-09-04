package db

import (
	"fmt"
)

// GetSubjectAliases retrieves all custom subject aliases for a profile
func (d *Database) GetSubjectAliases(profileID string) (map[string]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`SELECT original_subject, custom_alias FROM subject_aliases WHERE profile_id = ?`, profileID)
	if err != nil {
		return nil, fmt.Errorf("error querying subject aliases: %w", err)
	}
	defer rows.Close()

	aliases := make(map[string]string)
	for rows.Next() {
		var orig, alias string
		if err := rows.Scan(&orig, &alias); err != nil {
			continue
		}
		aliases[orig] = alias
	}
	return aliases, nil
}

// SetSubjectAlias creates or updates a custom subject alias
func (d *Database) SetSubjectAlias(profileID, originalSubject, customAlias string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `
		INSERT INTO subject_aliases (profile_id, original_subject, custom_alias)
		VALUES (?, ?, ?)
		ON CONFLICT(profile_id, original_subject) DO UPDATE SET custom_alias = excluded.custom_alias
	`
	_, err := d.db.Exec(query, profileID, originalSubject, customAlias)
	return err
}

// DeleteSubjectAlias removes a custom subject alias
func (d *Database) DeleteSubjectAlias(profileID, originalSubject string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`DELETE FROM subject_aliases WHERE profile_id = ? AND original_subject = ?`, profileID, originalSubject)
	return err
}
