package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// ColumnRule represents column-level mock/generation logic.
type ColumnRule struct {
	TruePercentage  float32         `json:"truePercentage,omitempty"`
	FalsePercentage float32         `json:"falsePercentage,omitempty"`
	NullPercentage  float32         `json:"nullPercentage,omitempty"`
	Pattern         string          `json:"pattern,omitempty"`
	Spec            json.RawMessage `json:"spec,omitempty"`
}

// TableRule represents table-level generation rules.
type TableRule struct {
	Count   int64                 `json:"count,omitempty"`
	Columns map[string]ColumnRule `json:"columns,omitempty"`
}

// FileConfig maps directly to the structure of your JSON rule file.
type FileConfig struct {
	Version string               `json:"version"`
	Ignore  []string             `json:"ignore,omitempty"`
	Tables  map[string]TableRule `json:"tables"`
}

// Config holds runtime options alongside parsed JSON rules.
type Config struct {
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

	return nil
}

func (c *Config) TableRule(table string) *TableRule {
	if table != "" {
		rule := c.Rules.Tables[table]
		return &rule
	}
	return nil
}
