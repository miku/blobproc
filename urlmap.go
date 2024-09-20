package blobproc

import (
	"sync"

	"github.com/jmoiron/sqlx"
)

// URLMap wraps an sqlite3 database for URL and SHA1 lookups.
type URLMap struct {
	Path string
	mu   sync.Mutex
	db   *sqlx.DB
}

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

}

func (u *URLMap) Insert(url, sha1 string) error {
}
