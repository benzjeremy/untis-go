package db

import (
	"fmt"
	"time"
)

// GetHomeworks retrieves all homework for the specified profile, ordered by due_date
func (d *Database) GetHomeworks(profileID string) ([]Homework, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, profile_id, subject, description, due_date, completed, created_at
		FROM homework
		WHERE profile_id = ?
		ORDER BY completed ASC, due_date ASC, created_at DESC
	`, profileID)
	if err != nil {
		return nil, fmt.Errorf("error querying homework: %w", err)
	}
	defer rows.Close()

	var list []Homework
	for rows.Next() {
		var h Homework
		var compInt int
		var createdAtStr string

		if err := rows.Scan(&h.ID, &h.ProfileID, &h.Subject, &h.Description, &h.DueDate, &compInt, &createdAtStr); err != nil {
			return nil, fmt.Errorf("error scanning homework: %w", err)
		}

		h.Completed = compInt == 1
		h.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if h.CreatedAt.IsZero() {
			h.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		}
		h.Source = "local"
		list = append(list, h)
	}

	return list, nil
}

// CreateHomework inserts a new homework entry into SQLite
func (d *Database) CreateHomework(h *Homework) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if h.ID == "" {
		h.ID = fmt.Sprintf("hw_%d", time.Now().UnixNano())
	}

	compInt := 0
	if h.Completed {
		compInt = 1
	}

	query := `
		INSERT INTO homework (id, profile_id, subject, description, due_date, completed, created_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			subject = excluded.subject,
			description = excluded.description,
			due_date = excluded.due_date,
			completed = excluded.completed
	`

	_, err := d.db.Exec(query, h.ID, h.ProfileID, h.Subject, h.Description, h.DueDate, compInt)
	return err
}

// UpdateHomeworkCompleted toggles or sets the completed flag of a homework entry
func (d *Database) UpdateHomeworkCompleted(id string, completed bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	compInt := 0
	if completed {
		compInt = 1
	}

	_, err := d.db.Exec(`UPDATE homework SET completed = ? WHERE id = ?`, compInt, id)
	return err
}

// DeleteHomework removes a homework entry from SQLite
func (d *Database) DeleteHomework(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`DELETE FROM homework WHERE id = ?`, id)
	return err
}
