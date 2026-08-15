package updater

import (
	"database/sql"
	"fmt"
	"log"
	"path"
	"sync"

	_ "modernc.org/sqlite"
)

const DATABASE_FILE_NAME = ".upstream-watch.sqlite"

type Database interface {
	AddEntry(Entry) error
	GetEntry(Entry) (Entry, error)
	GetLatestUpdatedEntry(moduleName string) (Entry, error)
}

type database struct {
	db    *sql.DB
	mutex sync.Mutex
}

type Entry struct {
	ID         int64
	ModuleName string
	Commit     string
	Updated    bool
}

const schema = `CREATE TABLE modules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name text,
    git_commit text NULL,
    updated boolean,
	UNIQUE (name, git_commit));`

func NewDatabase(runDir string) Database {
	dbPath := path.Join(runDir, DATABASE_FILE_NAME)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalln(err)
	}

	// force a connection and test that it worked
	err = db.Ping()
	if err != nil {
		log.Fatalln(err)
	}

	_, err = db.Exec(schema)
	if err != nil {
		if !(err.Error() == "table modules already exists") {
			log.Fatalln(err)
		}

		// The table already existed under some schema, but we don't know
		// which one. Rather than guessing at what changed and trying to
		// migrate it, fail loudly here with a fix a user can act on: a
		// schema mismatch will otherwise resurface later as a much more
		// confusing "no such column" error from a random query.
		if err := verifyModulesSchema(db); err != nil {
			log.Fatalf("%s has an incompatible modules table (%v). Delete the file and restart upstream-watch to recreate it.", dbPath, err)
		}
	}

	return &database{
		db: db,
	}
}

// verifyModulesSchema checks that an existing modules table matches the
// columns this version of upstream-watch expects.
func verifyModulesSchema(db *sql.DB) error {
	_, err := db.Exec("SELECT id, name, git_commit, updated FROM modules LIMIT 1")
	return err
}

func (d *database) AddEntry(e Entry) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction %v", err)
	}

	_, err = tx.Exec(
		"INSERT INTO modules (name, git_commit, updated) VALUES (?, ?, ?)",
		e.ModuleName, e.Commit, e.Updated,
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert new entry %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit changes %v", err)
	}

	return nil
}

func (d *database) GetEntry(e Entry) (Entry, error) {
	entry := Entry{}
	row := d.db.QueryRow(
		"SELECT id, name, git_commit, updated FROM modules WHERE name = ? AND git_commit = ?",
		e.ModuleName, e.Commit,
	)
	err := row.Scan(&entry.ID, &entry.ModuleName, &entry.Commit, &entry.Updated)
	if err != nil {
		return entry, err
	}

	return entry, nil
}

// GetLatestUpdatedEntry returns the most recently recorded successful update
// for moduleName, i.e. the commit that was last applied. It returns
// sql.ErrNoRows if the module has never been successfully updated before.
func (d *database) GetLatestUpdatedEntry(moduleName string) (Entry, error) {
	entry := Entry{}
	row := d.db.QueryRow(
		"SELECT id, name, git_commit, updated FROM modules WHERE name = ? AND updated = true ORDER BY id DESC LIMIT 1",
		moduleName,
	)
	err := row.Scan(&entry.ID, &entry.ModuleName, &entry.Commit, &entry.Updated)
	if err != nil {
		return entry, err
	}

	return entry, nil
}
