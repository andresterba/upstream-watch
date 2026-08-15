package updater

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "modernc.org/sqlite"
)

func newMock() (*sql.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	return mockDB, mock
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
				mock.ExpectRollback()
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMock()
			d := &database{
				db: db,
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
				rows := sqlmock.NewRows([]string{"id", "name", "git_commit", "updated"}).
					AddRow(1, "test", "testabcdef", true)
				mock.ExpectQuery("SELECT id, name, git_commit, updated FROM modules WHERE name = ? AND git_commit = ?").
					WithArgs("test", "testabcdef").
					WillReturnRows(rows)
			},
			want:    Entry{ID: 1, ModuleName: "test", Commit: "testabcdef", Updated: true},
			wantErr: false,
		},
		{
			name: "should return sql.ErrNoRows when no entry matches",
			args: args{
				e: Entry{ModuleName: "missing", Commit: "nope"},
			},
			mockClosure: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "git_commit", "updated"})
				mock.ExpectQuery("SELECT id, name, git_commit, updated FROM modules WHERE name = ? AND git_commit = ?").
					WithArgs("missing", "nope").
					WillReturnRows(rows)
			},
			want:    Entry{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMock()
			d := &database{
				db: db,
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

func Test_database_GetLatestUpdatedEntry(t *testing.T) {
	type args struct {
		moduleName string
	}
	tests := []struct {
		name        string
		args        args
		mockClosure func(mock sqlmock.Sqlmock)
		want        Entry
		wantErr     bool
	}{
		{
			name: "should get the latest updated entry",
			args: args{
				moduleName: "test",
			},
			mockClosure: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "git_commit", "updated"}).
					AddRow(3, "test", "testabcdef", true)
				mock.ExpectQuery("SELECT id, name, git_commit, updated FROM modules WHERE name = ? AND updated = true ORDER BY id DESC LIMIT 1").
					WithArgs("test").
					WillReturnRows(rows)
			},
			want:    Entry{ID: 3, ModuleName: "test", Commit: "testabcdef", Updated: true},
			wantErr: false,
		},
		{
			name: "should return sql.ErrNoRows when the module was never updated",
			args: args{
				moduleName: "never-updated",
			},
			mockClosure: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "git_commit", "updated"})
				mock.ExpectQuery("SELECT id, name, git_commit, updated FROM modules WHERE name = ? AND updated = true ORDER BY id DESC LIMIT 1").
					WithArgs("never-updated").
					WillReturnRows(rows)
			},
			want:    Entry{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMock()
			d := &database{
				db: db,
			}
			tt.mockClosure(mock)

			got, err := d.GetLatestUpdatedEntry(tt.args.moduleName)
			if (err != nil) != tt.wantErr {
				t.Errorf("database.GetLatestUpdatedEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("database.GetLatestUpdatedEntry() = %v, want %v", got, tt.want)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

// TestVerifyModulesSchema checks that a modules table predating the id
// column is correctly flagged as incompatible, without upstream-watch
// trying to migrate it.
func TestVerifyModulesSchema(t *testing.T) {
	tests := []struct {
		name        string
		createTable string
		wantErr     bool
	}{
		{
			name:        "current schema is compatible",
			createTable: schema,
			wantErr:     false,
		},
		{
			name: "schema predating the id column is incompatible",
			createTable: `CREATE TABLE modules (
				name text,
				git_commit text NULL,
				updated boolean,
				PRIMARY KEY (name, git_commit));`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatalf("failed to open in-memory db: %v", err)
			}
			defer db.Close()

			if _, err := db.Exec(tt.createTable); err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			err = verifyModulesSchema(db)
			if (err != nil) != tt.wantErr {
				t.Errorf("verifyModulesSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDatabase_Integration exercises AddEntry, GetEntry and
// GetLatestUpdatedEntry together against a real SQLite connection, covering
// behavior sqlmock can't meaningfully simulate: the UNIQUE(name, git_commit)
// constraint and the ORDER BY id DESC row selection in
// GetLatestUpdatedEntry when multiple rows exist.
func TestDatabase_Integration(t *testing.T) {
	// SetMaxOpenConns(1) keeps every query on the same connection, since
	// each new connection to ":memory:" otherwise gets its own independent
	// database.
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	d := &database{db: db}

	if err := d.AddEntry(Entry{ModuleName: "modA", Commit: "commit1", Updated: true}); err != nil {
		t.Fatalf("AddEntry() error = %v", err)
	}
	if err := d.AddEntry(Entry{ModuleName: "modA", Commit: "commit2", Updated: false}); err != nil {
		t.Fatalf("AddEntry() error = %v", err)
	}
	if err := d.AddEntry(Entry{ModuleName: "modA", Commit: "commit3", Updated: true}); err != nil {
		t.Fatalf("AddEntry() error = %v", err)
	}

	t.Run("duplicate (name, git_commit) is rejected", func(t *testing.T) {
		if err := d.AddEntry(Entry{ModuleName: "modA", Commit: "commit1", Updated: true}); err == nil {
			t.Fatal("expected AddEntry() to fail on duplicate (name, git_commit), got nil error")
		}
	})

	t.Run("GetEntry finds an inserted entry", func(t *testing.T) {
		got, err := d.GetEntry(Entry{ModuleName: "modA", Commit: "commit2"})
		if err != nil {
			t.Fatalf("GetEntry() error = %v", err)
		}
		if got.ModuleName != "modA" || got.Commit != "commit2" || got.Updated {
			t.Fatalf("GetEntry() = %+v, want ModuleName=modA Commit=commit2 Updated=false", got)
		}
	})

	t.Run("GetLatestUpdatedEntry picks the most recent updated=true row", func(t *testing.T) {
		got, err := d.GetLatestUpdatedEntry("modA")
		if err != nil {
			t.Fatalf("GetLatestUpdatedEntry() error = %v", err)
		}
		if got.Commit != "commit3" {
			t.Fatalf("GetLatestUpdatedEntry() = %+v, want Commit=commit3 (most recent updated=true)", got)
		}
	})

	t.Run("GetLatestUpdatedEntry returns sql.ErrNoRows for an unknown module", func(t *testing.T) {
		_, err := d.GetLatestUpdatedEntry("unknown-module")
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("GetLatestUpdatedEntry() error = %v, want sql.ErrNoRows", err)
		}
	})
}
