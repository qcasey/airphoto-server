package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

// ServerType holds global state of the iMessage server
type ServerType struct {
	// Main map and submaps of parsed album data
	Albums map[string]*Album

	// Device Tokens for firebase messaging
	DeviceTokens []string
	useFirebase  bool

	// Required SQLite DB information
	DB         *sql.DB
	SQLiteFile string
	AssetsPath string
	Started    bool

	// DeterminedName is the user's name derived from comment's "IsMine" bool.
	DeterminedName string

	LastModified time.Time
	lock         sync.RWMutex
}

func (s *ServerType) openDB() {
	var err error
	server.DB, err = sql.Open("sqlite3", server.SQLiteFile)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}
}

func query(SQL string) (*sql.Rows, error) {
	log.Debug().Msg(SQL)

	// Open new connection to the DB if it's been closed.
	// Some connections are persistent, to speed up rapid fire queries
	if err := server.DB.Ping(); err != nil {
		log.Info().Msg("Creating new DB connection")
		server.openDB()
		defer server.DB.Close()
	}

	rows, err := server.DB.Query(SQL)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, fmt.Errorf("Empty asset rows, did the SQL query complete successfully?")
	}
	return rows, nil
}

func hasBeenModified() bool {
	// Get SQL file info
	info, err := os.Stat(server.SQLiteFile)
	if err != nil {
		panic(err.Error())
	}
	infoWal, errWal := os.Stat(server.SQLiteFile + "-wal")

	// Set old modified time for reference
	oldModifiedTime := server.LastModified

	// Check both modified times
	if info.ModTime().After(server.LastModified) {
		server.LastModified = info.ModTime()
	}
	if errWal == nil && infoWal.ModTime().After(server.LastModified) {
		server.LastModified = infoWal.ModTime()
	}

	return server.LastModified.After(oldModifiedTime)
}

func readBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		http.Error(w, "can't read body", http.StatusBadRequest)
		return nil, fmt.Errorf("Error reading body: %v", err)
	}

	// Put body back
	r.Body.Close() //  must close
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return body, nil
}
