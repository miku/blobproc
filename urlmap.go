package blobproc

import (
	"sync"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

const urlmapSchema = `
create table if not exists map (
	url  text not null,
	sha1 text not null,
	timestamp datetime default CURRENT_TIMESTAMP
);
create index if not exists index_url_sha1 on map(url, sha1);
`

// URLMap wraps an sqlite3 database for URL and SHA1 lookups.
type URLMap struct {
	Path string
	mu   sync.Mutex
	db   *sqlx.DB
}

// EnsureDB creates a new database with schema, if it is not already set up.
func (u *URLMap) EnsureDB() error {
	if u.db != nil {
		return nil
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	db, err := sqlx.Connect("sqlite", u.Path)
	if err != nil {
		return err
	}
	_, err = db.Exec(urlmapSchema)
	if err != nil {
		return err
	}
	u.db = db
	return nil
}

// Insert inserts a new pair into the database. We lock at the application
// level to avoid 'database is locked (5) (SQLITE_BUSY)'
func (u *URLMap) Insert(url, sha1 string) error {
	u.mu.Lock()
	_, err := u.db.Exec(`insert into map (url, sha1) values (?, ?)`, url, sha1)
	u.mu.Unlock()
	return err
}
