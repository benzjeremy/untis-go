//go:build (linux || darwin) && cgo

package db

import (
	_ "github.com/mattn/go-sqlite3"
)

const sqliteDriver = "sqlite3"
