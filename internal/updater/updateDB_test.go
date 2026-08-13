package updater

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func newMock() (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	return sqlxDB, mock
}

func Test_database_AddEntry(t *testing.T) {
	type args struct {
		e *Entry
	}
	tests := []struct {
		name        string
		args        args
		mockClosure func(mock sqlmock.Sqlmock)
		wantErr     bool
	}{
		{
			name: "should add new entry",
			args: args{
				e: &Entry{ModuleName: "test", Commit: "testabcdef", Updated: true},
			},
			mockClosure: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO modules (name, git_commit, updated) VALUES (?, ?, ?)`).
					WithArgs("test", "testabcdef", true).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "should fail and not commit if insert fails",
			args: args{
				e: &Entry{ModuleName: "test", Commit: "testabcdef", Updated: true},
			},
			mockClosure: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO modules (name, git_commit, updated) VALUES (?, ?, ?)`).
					WithArgs("test", "testabcdef", true).
					WillReturnError(fmt.Errorf("some error"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMock()
			d := &database{
				db:    db,
				mutex: &sync.Mutex{},
			}
			tt.mockClosure(mock)
			if err := d.AddEntry(*tt.args.e); (err != nil) != tt.wantErr {
				t.Errorf("database.AddEntry() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func Test_database_GetEntry(t *testing.T) {
	type args struct {
		e Entry
	}
	tests := []struct {
		name        string
		args        args
		mockClosure func(mock sqlmock.Sqlmock)
		want        Entry
		wantErr     bool
	}{
		{
			name: "should get a entry",
			args: args{
				e: Entry{ModuleName: "test", Commit: "testabcdef", Updated: true},
			},
			mockClosure: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"name", "git_commit", "updated"}).
					AddRow("test", "testabcdef", true)
				mock.ExpectQuery("SELECT * FROM modules WHERE name=$1 AND git_commit=$2").WithArgs("test", "testabcdef").WillReturnRows(rows)
			},
			want:    Entry{ModuleName: "test", Commit: "testabcdef", Updated: true},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMock()
			d := &database{
				db:    db,
				mutex: &sync.Mutex{},
			}
			tt.mockClosure(mock)

			got, err := d.GetEntry(tt.args.e)
			if (err != nil) != tt.wantErr {
				t.Errorf("database.GetEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("database.GetEntry() = %v, want %v", got, tt.want)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

// TestNewDatabase_MigratesLegacySchema verifies that a modules table created
// by a pre-migration version of upstream-watch (no id column, PRIMARY KEY
// (name, git_commit)) is upgraded in place: existing rows survive, and are
// assigned ids in their original insertion order so GetLatestUpdatedEntry
// keeps meaning "most recent" across the upgrade.
func TestNewDatabase_MigratesLegacySchema(t *testing.T) {
	dir := t.TempDir()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir into temp dir: %v", err)
	}
	defer os.Chdir(orig)

	legacyDB, err := sqlx.Connect("sqlite3", "./.upstream-watch.sqlite")
	if err != nil {
		t.Fatalf("failed to open legacy db: %v", err)
	}
	if _, err := legacyDB.Exec(`CREATE TABLE modules (
		name text,
		git_commit text NULL,
		updated boolean,
		PRIMARY KEY (name, git_commit));`); err != nil {
		t.Fatalf("failed to create legacy schema: %v", err)
	}
	if _, err := legacyDB.Exec(`INSERT INTO modules (name, git_commit, updated) VALUES
		('svc', 'commit-a', 1),
		('svc', 'commit-b', 1),
		('other', 'commit-x', 1)`); err != nil {
		t.Fatalf("failed to seed legacy rows: %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("failed to close legacy db: %v", err)
	}

	db := NewDatabase()

	got, err := db.GetEntry(Entry{ModuleName: "svc", Commit: "commit-a"})
	if err != nil {
		t.Fatalf("expected legacy row to survive migration: %v", err)
	}
	if !got.Updated {
		t.Errorf("expected migrated row to keep Updated=true, got %+v", got)
	}

	latest, err := db.GetLatestUpdatedEntry("svc")
	if err != nil {
		t.Fatalf("GetLatestUpdatedEntry failed after migration: %v", err)
	}
	if latest.Commit != "commit-b" {
		t.Errorf("expected latest entry to be commit-b (inserted last), got %q", latest.Commit)
	}

	// re-opening (and thus re-running the migration check) must be stable
	db2 := NewDatabase()
	latest2, err := db2.GetLatestUpdatedEntry("svc")
	if err != nil {
		t.Fatalf("GetLatestUpdatedEntry failed on re-open: %v", err)
	}
	if latest2.Commit != "commit-b" {
		t.Errorf("expected latest entry to remain commit-b on re-open, got %q", latest2.Commit)
	}

	if err := db2.AddEntry(Entry{ModuleName: "svc", Commit: "commit-c", Updated: true}); err != nil {
		t.Fatalf("AddEntry after migration failed: %v", err)
	}

	latest3, err := db2.GetLatestUpdatedEntry("svc")
	if err != nil {
		t.Fatalf("GetLatestUpdatedEntry failed after post-migration insert: %v", err)
	}
	if latest3.Commit != "commit-c" {
		t.Errorf("expected commit-c to be latest after insert, got %q", latest3.Commit)
	}
}
