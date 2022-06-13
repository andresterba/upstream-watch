package config

import (
	"reflect"
	"testing"
)

func TestGetUpdateConfig(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *UpdateConfig
		wantErr bool
	}{
		{
			name: "should parse update config",
			args: args{
				path: "testdata/update-config.yaml",
			},
			want: &UpdateConfig{
				PreUpdateCommand:  "ls",
				PostUpdateCommand: "docker ps",
			},
			wantErr: false,
		},
		{
			name: "should error on empty path",
			args: args{
				path: "not/exisiting/path/update-config.yaml",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetUpdateConfig(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUpdateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUpdateConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
