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
}

type database struct {
	db    *sqlx.DB
	mutex *sync.Mutex
}

type Entry struct {
	ModuleName string `db:"name"`
	Commit     string `db:"git_commit"`
	Updated    bool   `db:"updated"`
}

const schema = `CREATE TABLE modules (
    name text,
    git_commit text NULL,
    updated boolean,
	PRIMARY KEY (name, git_commit));`

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
	}

	return &database{
		db:    db,
		mutex: &sync.Mutex{},
	}
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
