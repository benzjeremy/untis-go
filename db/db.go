package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// Database wraps the sql.DB instance and provides concurrency-safe access
type Database struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

// GetDBPath returns the standard path to ~/.local/share/untis-go/untis.db
func GetDBPath() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "untis-go", "untis.db")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "untis-go", "untis.db")
}

// InitDB initializes SQLite database at the specified path (or standard path if empty)
func InitDB(customPath ...string) (*Database, error) {
	dbPath := GetDBPath()
	if len(customPath) > 0 && customPath[0] != "" {
		dbPath = customPath[0]
	}

	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("could not create database directory %s: %w", dir, err)
	}

	// Connect to SQLite with WAL mode & busy timeout
	connStr := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON&_synchronous=NORMAL", dbPath)
	sqldb, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("could not open sqlite database at %s: %w", dbPath, err)
	}

	// Configure pool for concurrency
	sqldb.SetMaxOpenConns(25)
	sqldb.SetMaxIdleConns(10)

	db := &Database{
		db:   sqldb,
		path: dbPath,
	}

	// Create tables & indexes
	if err := db.createSchema(); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("failed to create database schema: %w", err)
	}

	// Automatically run legacy migration if profiles table is empty
	if err := db.checkAndMigrate(); err != nil {
		log.Printf("[DB Migration] Hinweis bei Migration: %v", err)
	}

	return db, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// GetUnderlyingDB returns the raw *sql.DB if needed
func (d *Database) GetUnderlyingDB() *sql.DB {
	return d.db
}

// createSchema initializes all required tables and indexes
func (d *Database) createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS profiles (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		school TEXT NOT NULL,
		server TEXT NOT NULL,
		username TEXT NOT NULL,
		encrypted_password TEXT NOT NULL,
		is_active INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS classes (
		id INTEGER NOT NULL,
		school TEXT NOT NULL,
		name TEXT NOT NULL,
		long_name TEXT,
		active INTEGER NOT NULL DEFAULT 1,
		PRIMARY KEY (school, id)
	);

	CREATE TABLE IF NOT EXISTS timetable_cache (
		class_id INTEGER NOT NULL,
		date TEXT NOT NULL,
		data_json TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (class_id, date)
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS homework (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL,
		subject TEXT NOT NULL,
		description TEXT NOT NULL,
		due_date TEXT NOT NULL,
		completed INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS absences (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL,
		reason TEXT NOT NULL,
		text TEXT NOT NULL,
		start_date TEXT NOT NULL,
		end_date TEXT NOT NULL,
		is_excused INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_classes_school ON classes(school);
	CREATE INDEX IF NOT EXISTS idx_tt_cache ON timetable_cache(class_id, date);
	CREATE INDEX IF NOT EXISTS idx_profiles_active ON profiles(is_active);
	CREATE INDEX IF NOT EXISTS idx_homework_profile ON homework(profile_id);
	CREATE INDEX IF NOT EXISTS idx_absences_profile ON absences(profile_id);
	`

	_, err := d.db.Exec(schema)
	return err
}
