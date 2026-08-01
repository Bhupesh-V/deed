package models

import "strings"

type Entity struct {
	Schema  string
	Name    string
	Columns []Column
	// TODO: tables can have Constraint too!
	// Constraints []Constraint
}

type Column struct {
	Name       string
	Type       DataType
	Constraint []Constraint
}

// DataType captures the full PostgreSQL type signature.
type DataType struct {
	BaseType  string // e.g., "VARCHAR", "NUMERIC", "INT"
	Precision *int   // e.g., 10 for NUMERIC(10,2) or 255 for VARCHAR(255)
	Scale     *int   // e.g., 2 for NUMERIC(10,2)
}

type Constraint struct {
	Name             string
	Type             string
	ReferencedTable  *string
	ReferencedColumn *string
}

type ForeignKeyDetails struct {
	ParentTable   string
	ColumnMapping map[string]string // local_column -> parent_column
	OnDelete      string            // e.g., "CASCADE", "SET NULL"
	OnUpdate      string            // e.g., "RESTRICT"
}

// GetFK returns Foreign Key (FK) relationship details directly from a column's constraints
// TODO: handle composite FKs
func (c *Column) GetFK() (string, string, bool) {
	for _, ctr := range c.Constraint {
		if ctr.Type == ForeignKey.String() && ctr.ReferencedTable != nil && *ctr.ReferencedTable != "" {
			// TODO: unreliable assumption!!!!
			refCol := "id"
			if ctr.ReferencedColumn != nil && *ctr.ReferencedColumn != "" {
				refCol = *ctr.ReferencedColumn
			}
			return *ctr.ReferencedTable, refCol, true
		}
	}
	return "", "", false
}

func (c *Column) IsAutoIncrement() bool {
	dt := strings.ToLower(c.Type.BaseType)
	return strings.Contains(dt, "serial")
}

// GenerationRule defines how a specific column generates fake data.
type GenerationRule struct {
	Type         string // "regex", "random_string", "random_int", "email", "foreign_key", "custom"
	Min          int
	Max          int
	Prefix       string
	RegexPattern string
	CustomSQL    string
}
