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
	mutex *sync.Mutex
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
const schema = `CREATE TABLE modules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name text,
    git_commit text NULL,
    updated boolean,
	UNIQUE (name, git_commit));`

func NewDatabase() Database {

	// this Pings the database trying to connect
	// use sqlx.Open() for sql.Open() semantics
	db, err := sqlx.Connect("sqlite3", "./.upstream-watch.sqlite")
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

		if err := migrateModulesTableToIDColumn(db); err != nil {
			log.Fatalln(err)
		}
	}

	return &database{
		db:    db,
		mutex: &sync.Mutex{},
	}
}

// migrateModulesTableToIDColumn upgrades a modules table created by a
// version of upstream-watch that predates the explicit id column (i.e. one
// keyed only by PRIMARY KEY (name, git_commit)). It is a no-op if the table
// already has an id column. Existing rows are preserved, and are given ids
// in their original insertion order so GetLatestUpdatedEntry's ordering
// keeps meaning "most recent" across the upgrade.
func migrateModulesTableToIDColumn(db *sqlx.DB) error {
	hasIDColumn, err := modulesTableHasIDColumn(db)
	if err != nil {
		return fmt.Errorf("failed to inspect modules table: %w", err)
	}

	if hasIDColumn {
		return nil
	}

	log.Println("migrating modules table to add an explicit id column")

	tx := db.MustBegin()
	defer tx.Rollback()

	if _, err := tx.Exec("ALTER TABLE modules RENAME TO modules_pre_id_migration"); err != nil {
		return fmt.Errorf("failed to rename old modules table: %w", err)
	}

	if _, err := tx.Exec(schema); err != nil {
		return fmt.Errorf("failed to create new modules table: %w", err)
	}

	if _, err := tx.Exec(`INSERT INTO modules (name, git_commit, updated)
		SELECT name, git_commit, updated FROM modules_pre_id_migration ORDER BY rowid`); err != nil {
		return fmt.Errorf("failed to copy rows into migrated modules table: %w", err)
	}

	if _, err := tx.Exec("DROP TABLE modules_pre_id_migration"); err != nil {
		return fmt.Errorf("failed to drop pre-migration modules table: %w", err)
	}

	return tx.Commit()
}

func modulesTableHasIDColumn(db *sqlx.DB) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(modules)")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notNull   int
			dfltValue any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dfltValue, &pk); err != nil {
			return false, err
		}

		if name == "id" {
			return true, nil
		}
	}

	return false, rows.Err()
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
