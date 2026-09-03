package db

import (
	"time"
)

// Profile models a WebUntis user account in the SQLite 'profiles' table
type Profile struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	School            string    `json:"school"`
	Server            string    `json:"server"`
	Username          string    `json:"username"`
	EncryptedPassword string    `json:"-"`
	Password          string    `json:"password,omitempty"` // decrypted in-memory only
	IsActive          bool      `json:"isActive"`
	CreatedAt         time.Time `json:"createdAt"`
}

// Class models an entry in the SQLite 'classes' table
type Class struct {
	ID       int    `json:"id"`
	School   string `json:"school"`
	Name     string `json:"name"`
	LongName string `json:"longName"`
	Active   bool   `json:"active"`
}

// TimetableCacheEntry represents a cached day's timetable in SQLite
type TimetableCacheEntry struct {
	ClassID   int       `json:"classId"`
	Date      string    `json:"date"` // Format "YYYY-MM-DD"
	DataJSON  string    `json:"dataJson"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Setting represents a key-value pair in SQLite
type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
