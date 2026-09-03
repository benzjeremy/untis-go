package db

import (
	"fmt"
	"time"
)

// GetAbsences retrieves all recorded absences for the specified profile, newest first
func (d *Database) GetAbsences(profileID string) ([]Absence, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, profile_id, reason, text, start_date, end_date, is_excused, created_at
		FROM absences
		WHERE profile_id = ?
		ORDER BY start_date DESC, created_at DESC
	`, profileID)
	if err != nil {
		return nil, fmt.Errorf("error querying absences: %w", err)
	}
	defer rows.Close()

	var list []Absence
	for rows.Next() {
		var a Absence
		var excInt int
		var createdAtStr string

		if err := rows.Scan(&a.ID, &a.ProfileID, &a.Reason, &a.Text, &a.StartDate, &a.EndDate, &excInt, &createdAtStr); err != nil {
			return nil, fmt.Errorf("error scanning absence: %w", err)
		}

		a.IsExcused = excInt == 1
		a.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if a.CreatedAt.IsZero() {
			a.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		}
		a.Source = "local"
		list = append(list, a)
	}

	return list, nil
}

// CreateAbsence inserts a new absence entry into SQLite
func (d *Database) CreateAbsence(a *Absence) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if a.ID == "" {
		a.ID = fmt.Sprintf("abs_%d", time.Now().UnixNano())
	}

	excInt := 0
	if a.IsExcused {
		excInt = 1
	}

	query := `
		INSERT INTO absences (id, profile_id, reason, text, start_date, end_date, is_excused, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			reason = excluded.reason,
			text = excluded.text,
			start_date = excluded.start_date,
			end_date = excluded.end_date,
			is_excused = excluded.is_excused
	`

	_, err := d.db.Exec(query, a.ID, a.ProfileID, a.Reason, a.Text, a.StartDate, a.EndDate, excInt)
	return err
}

// DeleteAbsence removes an absence record from SQLite
func (d *Database) DeleteAbsence(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`DELETE FROM absences WHERE id = ?`, id)
	return err
}
