package updater

import (
	"fmt"
	"log"
	"sync"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type Database interface {
	AddEntry(Entry) error
	GetEntry(Entry) (Entry, error)
	GetLatestUpdatedEntry(moduleName string) (Entry, error)
}

type database struct {
	db    *sqlx.DB
	mutex sync.Mutex
}

type Entry struct {
	ID         int64  `db:"id"`
	ModuleName string `db:"name"`
	Commit     string `db:"git_commit"`
	Updated    bool   `db:"updated"`
}

// id is an explicit, self-documenting monotonically increasing column used
// to order entries by recency (see GetLatestUpdatedEntry). It intentionally
// does not rely on SQLite's implicit rowid, since that's a driver/engine
// implementation detail rather than a guarantee we want this query to lean
// on.
const dbPath = "./.upstream-watch.sqlite"

const schema = `CREATE TABLE modules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name text,
    git_commit text NULL,
    updated boolean,
	UNIQUE (name, git_commit));`

func NewDatabase() Database {

	// this Pings the database trying to connect
	// use sqlx.Open() for sql.Open() semantics
	db, err := sqlx.Connect("sqlite3", dbPath)
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
func verifyModulesSchema(db *sqlx.DB) error {
	_, err := db.Exec("SELECT id, name, git_commit, updated FROM modules LIMIT 1")
	return err
}

func (d *database) AddEntry(e Entry) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	tx := d.db.MustBegin()
	_, err := tx.NamedExec(
		"INSERT INTO modules (name, git_commit, updated) VALUES (:name, :git_commit, :updated)",
		e,
	)
	if err != nil {
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
	err := d.db.Get(&entry, "SELECT * FROM modules WHERE name=$1 AND git_commit=$2", e.ModuleName, e.Commit)
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
	err := d.db.Get(&entry, "SELECT * FROM modules WHERE name=$1 AND updated=true ORDER BY id DESC LIMIT 1", moduleName)
	if err != nil {
		return entry, err
	}

	return entry, nil
}
