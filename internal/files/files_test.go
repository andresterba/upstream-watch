package files

import (
	"reflect"
	"testing"
)

func TestNewDirectoryScanner(t *testing.T) {
	type args struct {
		directoriesToIgnore []string
	}
	tests := []struct {
		name string
		args args
		want *DirectoryScanner
	}{
		{
			name: "should create a new directory scanner",
			args: args{
				directoriesToIgnore: []string{},
			},
			want: &DirectoryScanner{
				directoriesToIgnore: make(map[string]struct{}),
			},
		},
		{
			name: "should create a new directory scanner with directories to ignore",
			args: args{
				directoriesToIgnore: []string{"test"},
			},
			want: &DirectoryScanner{
				directoriesToIgnore: map[string]struct{}{"test": {}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDirectoryScanner(tt.args.directoriesToIgnore); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDirectoryScanner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirectoryScanner_ListDirectories(t *testing.T) {
	type fields struct {
		directoriesToIgnore map[string]struct{}
	}
	tests := []struct {
		name    string
		fields  fields
		want    []string
		wantErr bool
	}{
		{
			name: "should list all directories",
			fields: fields{
				directoriesToIgnore: map[string]struct{}{"": {}},
			},
			want:    []string{"testdata"},
			wantErr: false,
		},
		{
			name: "should not list ignored directories",
			fields: fields{
				directoriesToIgnore: map[string]struct{}{"testdata": {}},
			},
			want:    []string{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &DirectoryScanner{
				directoriesToIgnore: tt.fields.directoriesToIgnore,
			}
			got, err := ds.ListDirectories(".")
			if (err != nil) != tt.wantErr {
				t.Errorf("DirectoryScanner.ListDirectories() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DirectoryScanner.ListDirectories() = %v, want %v", got, tt.want)
			}
		})
	}
}
