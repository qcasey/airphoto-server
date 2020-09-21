package database

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	// Required SQLite DB information
	DB   *sql.DB
	File string

	LastModified time.Time
	lock         sync.RWMutex
)

func Open() error {
	var err error
	DB, err = sql.Open("sqlite3", File)
	return err
}

func Query(SQL string) (*sql.Rows, error) {
	// Open new connection to the DB if it's been closed.
	// Some connections are persistent, to speed up rapid fire queries
	if err := DB.Ping(); err != nil {
		Open()
		defer DB.Close()
	}

	rows, err := DB.Query(SQL)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, fmt.Errorf("Empty asset rows, did the SQL query complete successfully?")
	}
	return rows, nil
}

func HasBeenModified() bool {
	// Get SQL file info
	info, err := os.Stat(File)
	if err != nil {
		// Probably shouldn't panic here, but if the DB file is gone there's a larger issue at play
		panic(err.Error())
	}
	infoWal, errWal := os.Stat(File + "-wal")

	// Set old modified time for reference
	oldModifiedTime := LastModified

	// Check both modified times
	if info.ModTime().After(LastModified) {
		LastModified = info.ModTime()
	}
	if errWal == nil && infoWal.ModTime().After(LastModified) {
		LastModified = infoWal.ModTime()
	}

	return LastModified.After(oldModifiedTime)
}
