package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// GetClasses returns all cached classes for a specific school
func (d *Database) GetClasses(school string) ([]Class, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, school, name, COALESCE(long_name, ''), active 
		FROM classes 
		WHERE school = ? 
		ORDER BY name ASC
	`, school)
	if err != nil {
		return nil, fmt.Errorf("error querying classes: %w", err)
	}
	defer rows.Close()

	var classes []Class
	for rows.Next() {
		var c Class
		var activeInt int
		if err := rows.Scan(&c.ID, &c.School, &c.Name, &c.LongName, &activeInt); err != nil {
			return nil, err
		}
		c.Active = activeInt == 1
		classes = append(classes, c)
	}

	return classes, nil
}

// GetClassByID returns a specific class for a school by ID
func (d *Database) GetClassByID(school string, id int) (*Class, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	row := d.db.QueryRow(`
		SELECT id, school, name, COALESCE(long_name, ''), active 
		FROM classes 
		WHERE school = ? AND id = ?
	`, school, id)

	var c Class
	var activeInt int
	if err := row.Scan(&c.ID, &c.School, &c.Name, &c.LongName, &activeInt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	c.Active = activeInt == 1
	return &c, nil
}

// SaveClasses updates or inserts classes in a fast batch transaction
func (d *Database) SaveClasses(school string, classes []Class) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(classes) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO classes (id, school, name, long_name, active)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(school, id) DO UPDATE SET
			name = excluded.name,
			long_name = excluded.long_name,
			active = excluded.active
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range classes {
		activeInt := 1
		if !c.Active {
			activeInt = 0
		}
		if _, err := stmt.Exec(c.ID, school, c.Name, c.LongName, activeInt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SearchClasses returns classes matching a filter substring
func (d *Database) SearchClasses(school, term string) ([]Class, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	pattern := "%" + strings.TrimSpace(term) + "%"
	rows, err := d.db.Query(`
		SELECT id, school, name, COALESCE(long_name, ''), active 
		FROM classes 
		WHERE school = ? AND (name LIKE ? OR long_name LIKE ?) 
		ORDER BY name ASC
	`, school, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var classes []Class
	for rows.Next() {
		var c Class
		var activeInt int
		if err := rows.Scan(&c.ID, &c.School, &c.Name, &c.LongName, &activeInt); err != nil {
			return nil, err
		}
		c.Active = activeInt == 1
		classes = append(classes, c)
	}

	return classes, nil
}
