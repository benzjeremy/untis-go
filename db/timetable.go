package db

import (
	"database/sql"
	"fmt"
	"time"
)

// GetTimetableCache retrieves the cached timetable JSON for a single date
func (d *Database) GetTimetableCache(classID int, date string) (string, time.Time, bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var dataJSON string
	var updatedAtStr string

	err := d.db.QueryRow(`
		SELECT data_json, updated_at 
		FROM timetable_cache 
		WHERE class_id = ? AND date = ?
	`, classID, date).Scan(&dataJSON, &updatedAtStr)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", time.Time{}, false, nil
		}
		return "", time.Time{}, false, err
	}

	updatedAt, _ := time.Parse("2006-01-02 15:04:05", updatedAtStr)
	if updatedAt.IsZero() {
		updatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	}

	return dataJSON, updatedAt, true, nil
}

// GetTimetableCacheRange retrieves all cached days within [startDate, endDate] (inclusive)
func (d *Database) GetTimetableCacheRange(classID int, startDate, endDate string) ([]TimetableCacheEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT class_id, date, data_json, updated_at 
		FROM timetable_cache 
		WHERE class_id = ? AND date >= ? AND date <= ? 
		ORDER BY date ASC
	`, classID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("error querying timetable cache range: %w", err)
	}
	defer rows.Close()

	var entries []TimetableCacheEntry
	for rows.Next() {
		var e TimetableCacheEntry
		var updatedAtStr string

		if err := rows.Scan(&e.ClassID, &e.Date, &e.DataJSON, &updatedAtStr); err != nil {
			return nil, err
		}

		e.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if e.UpdatedAt.IsZero() {
			e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		}

		entries = append(entries, e)
	}

	return entries, nil
}

// SaveTimetableCache saves or updates a cached day in SQLite
func (d *Database) SaveTimetableCache(classID int, date string, dataJSON string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `
		INSERT INTO timetable_cache (class_id, date, data_json, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(class_id, date) DO UPDATE SET
			data_json = excluded.data_json,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := d.db.Exec(query, classID, date, dataJSON)
	return err
}

// SaveTimetableCacheBatch saves multiple timetable days in a single fast transaction
func (d *Database) SaveTimetableCacheBatch(entries []TimetableCacheEntry) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(entries) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO timetable_cache (class_id, date, data_json, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(class_id, date) DO UPDATE SET
			data_json = excluded.data_json,
			updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range entries {
		if _, err := stmt.Exec(e.ClassID, e.Date, e.DataJSON); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ClearTimetableCache clears timetable cache for a specific class, or all classes if not specified
func (d *Database) ClearTimetableCache(classID ...int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(classID) > 0 && classID[0] > 0 {
		_, err := d.db.Exec(`DELETE FROM timetable_cache WHERE class_id = ?`, classID[0])
		return err
	}

	_, err := d.db.Exec(`DELETE FROM timetable_cache`)
	return err
}

// FindCachedLessonsRange returns all data_json strings for the given date range across all classes
func (d *Database) FindCachedLessonsRange(startDate, endDate string) ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT data_json 
		FROM timetable_cache 
		WHERE date >= ? AND date <= ?
	`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blobs []string
	for rows.Next() {
		var j string
		if err := rows.Scan(&j); err == nil {
			blobs = append(blobs, j)
		}
	}
	return blobs, nil
}

