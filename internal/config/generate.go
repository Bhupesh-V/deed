package config

import (
	"deed/internal/models"
	"encoding/json"
	"slices"
	"strings"
)

const (
	defaultNullPercentage  float32 = 10
	defaultTruePercentage  float32 = 50
	defaultFalsePercentage float32 = 50
)

// autoIgnoreTables lists tables that are excluded from generation by default
var autoIgnoreTables = []string{"schema_migrations"}

// GenerateFromEntities builds a starter FileConfig by inspecting a live
// database schema. Each ColumnRule only contains the keys relevant to that column's type
func GenerateFromEntities(entities []models.Entity) FileConfig {
	tables := make(map[string]TableRule, len(entities))
	var ignore []string

	for _, e := range entities {
		if slices.Contains(autoIgnoreTables, e.Name) {
			ignore = append(ignore, e.Name)
			continue
		}

		var columns map[string]ColumnRule

		for _, col := range e.Columns {
			rule := columnRuleFor(col)
			if rule.isEmpty() {
				continue
			}

			if columns == nil {
				columns = make(map[string]ColumnRule)
			}
			columns[col.Name] = rule
		}

		tables[e.Name] = TableRule{Columns: columns}
	}

	return FileConfig{
		Version: "1",
		Ignore:  ignore,
		Tables:  tables,
	}
}

func columnRuleFor(col models.Column) ColumnRule {
	var rule ColumnRule

	if col.Nullable {
		rule.NullPercentage = defaultNullPercentage
	}

	switch strings.ToLower(col.Type.BaseType) {
	case "bool":
		rule.TruePercentage = defaultTruePercentage
		rule.FalsePercentage = defaultFalsePercentage

	case "json", "jsonb":
		rule.Spec = json.RawMessage("{}")
	}

	return rule
}

func (r ColumnRule) isEmpty() bool {
	return r.NullPercentage == 0 &&
		r.TruePercentage == 0 &&
		r.FalsePercentage == 0 &&
		r.Pattern == "" &&
		len(r.Spec) == 0
}
