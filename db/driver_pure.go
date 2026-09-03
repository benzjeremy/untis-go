//go:build windows || !cgo

package db

import (
	_ "modernc.org/sqlite"
)

const sqliteDriver = "sqlite"
