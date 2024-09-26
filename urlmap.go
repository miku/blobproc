package blobproc

import (
	"sync"

	"github.com/jmoiron/sqlx"
)

// URLMap wraps an sqlite3 database for URL and SHA1 lookups.
type URLMap struct {
	Path string // location of the database
	mu   sync.Mutex
	db   *sqlx.DB
}

// TODO: init simple k-v schema w/ indices

func (u *URLMap) ensureDB() error {
	if u.db != nil {
		return nil
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	db, err := sqlx.Connect("sqlite", u.Path)
	if err != nil {
		return err
	}
	u.db = db
	return nil
}

func (u *URLMap) Insert(url, sha1 string) error {
	if err := u.ensureDB(); err != nil {
		return err
	}
	db.Exec(`insert into map (url, sha1) values (?, ?)`, url, sha1)
	return nil
}
