package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// ColumnRule represents column-level mock/generation logic.
type ColumnRule struct {
	Type    string `json:"type"`
	Pattern string `json:"pattern,omitempty"`
}

// TableRule represents table-level generation rules.
type TableRule struct {
	Count   int64                 `json:"count"`
	Columns map[string]ColumnRule `json:"columns"`
}

// DatabaseConfig contains target database engine settings.
type DatabaseConfig struct {
	Name string `json:"name"`
}

// RulesConfig wraps table and ignore settings.
type RulesConfig struct {
	IgnoreTables []string             `json:"ignore_tables"`
	Tables       map[string]TableRule `json:"tables"`
}

// FileConfig maps directly to the structure of your JSON rule file.
type FileConfig struct {
	Version  string         `json:"version"`
	Database DatabaseConfig `json:"database"`
	Rules    RulesConfig    `json:"rules"`
}

// Config holds runtime options alongside parsed JSON rules.
type Config struct {
	Name  string
	Rules FileConfig
}

func New() *Config {
	return &Config{}
}

// LoadFromFile parses the JSON configuration file at the provided path into the Config struct.
func (c *Config) LoadFromFile(path string) error {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	var fileCfg FileConfig
	if err := json.Unmarshal(fileBytes, &fileCfg); err != nil {
		return fmt.Errorf("failed to parse config JSON: %w", err)
	}

	c.Rules = fileCfg
	if c.Name == "" {
		c.Name = fileCfg.Database.Name
	}

	return nil
}
