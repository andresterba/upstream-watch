package updater

import (
	"fmt"
	"log"
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
