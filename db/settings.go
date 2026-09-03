package db

import (
	"fmt"
	"strconv"
)

// GetSetting retrieves a string setting from SQLite, or returns defaultValue if not found
func (d *Database) GetSetting(key, defaultValue string) string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var val string
	err := d.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&val)
	if err != nil {
		return defaultValue
	}
	return val
}

// SetSetting saves or updates a key-value setting in SQLite
func (d *Database) SetSetting(key, value string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`
	_, err := d.db.Exec(query, key, value)
	return err
}

// GetIntSetting retrieves an integer setting, or returns defaultValue if missing/invalid
func (d *Database) GetIntSetting(key string, defaultValue int) int {
	str := d.GetSetting(key, "")
	if str == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}
	return val
}

// SetIntSetting stores an integer setting in SQLite
func (d *Database) SetIntSetting(key string, value int) error {
	return d.SetSetting(key, strconv.Itoa(value))
}

// GetAllSettings retrieves all settings as a key-value map
func (d *Database) GetAllSettings() (map[string]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, fmt.Errorf("error querying settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		settings[k] = v
	}

	return settings, nil
}
