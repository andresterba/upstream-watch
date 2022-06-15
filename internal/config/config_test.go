package config

import (
	"reflect"
	"testing"
)

func TestGetConfig(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name: "should parse config",
			args: args{
				path: "testdata/config.yaml",
			},
			want: &Config{
				RetryIntervall: 1337,
				IgnoreFolders:  []string{".git", ".test"},
			},
			wantErr: false,
		},
		{
			name: "should error on empty path",
			args: args{
				path: "not/exisiting/path/config.yaml",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetConfig(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
