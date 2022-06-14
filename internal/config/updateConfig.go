package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type UpdateConfig struct {
	PreUpdateCommands  []string `yaml:"pre_update_commands,omitempty"`
	PostUpdateCommands []string `yaml:"post_update_commands,omitempty"`
}

func GetUpdateConfig(path string) (*UpdateConfig, error) {
	updateConfig := UpdateConfig{}

	readConfig, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read update config for %s %v", path, err)
	}

	err = yaml.Unmarshal(readConfig, &updateConfig)
	if err != nil {
		return nil, fmt.Errorf("could not read update config dor %s %v", path, err)
	}

	return &updateConfig, nil
}
