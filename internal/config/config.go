package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	SingleDirectoryMode bool          `yaml:"single_directory_mode,omitempty"`
	RetryIntervall      time.Duration `yaml:"retry_intervall,omitempty"`
	IgnoreFolders       []string      `yaml:"ignore_folders,omitempty"`
}

func GetConfig(path string) (*Config, error) {
	config := Config{}

	readConfig, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config %v", err)
	}

	err = yaml.Unmarshal(readConfig, &config)
	if err != nil {
		return nil, fmt.Errorf("could not read config %v", err)
	}

	return &config, nil
}
