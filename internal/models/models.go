package models

import (
	"slices"
	"strings"
)

type Entity struct {
	Schema  string
	Name    string
	Columns []Column
	// TODO: tables can have Constraint too!
	// Constraints []Constraint
}

type Column struct {
	Name         string
	Type         DataType
	Constraint   []Constraint
	Nullable     bool
	Default      *string
	HasIdentity  bool
	IsPrimaryKey bool
}

// DataType captures the full PostgreSQL type signature.
type DataType struct {
	BaseType  string // e.g., "VARCHAR", "NUMERIC", "INT"
	Precision *int   // e.g., 10 for NUMERIC(10,2) or 255 for VARCHAR(255)
	Scale     *int   // e.g., 2 for NUMERIC(10,2)
	Radix     *int
	Length    *int32
}

type Constraint struct {
	Name             string
	Type             string
	ReferencedTable  *string
	ReferencedColumn *string
}

// FK returns Foreign Key (FK) relationship details directly from a column's constraints
// TODO: handle composite FKs
func (c *Column) FK() (parentTable string, ok bool) {
	for _, ctr := range c.Constraint {
		if ctr.Type == ForeignKey.String() && ctr.ReferencedTable != nil && *ctr.ReferencedTable != "" {
			return *ctr.ReferencedTable, true
		}
	}
	return "", false
}

func (c *Column) IsAutoIncrement() bool {
	dt := strings.ToLower(c.Type.BaseType)

	if c.IsPrimaryKey && c.Default != nil {
		return true
	} else if c.HasIdentity {
		return true
	} else {
		strings.Contains(dt, "serial")
	}
	return false
}

func (c *Column) HasUniqueConstraint() bool {
	for _, c := range c.Constraint {
		if c.Type == Unique.String() {
			return true
		}
	}
	return false
}

// PK returns Primary Key column for an Entity
func (e *Entity) PK() *Column {
	for _, c := range e.Columns {
		if c.IsPrimaryKey {
			return &c
		}
	}
	return nil
}

// IsOrdered validates whether the column can be sorted (i.e ordered) based on its data type.
func (c *Column) IsOrdered() bool {
	return slices.Contains([]string{"int", "int8", "int4", "serial"}, c.Type.BaseType)
}

func (e *Entity) DirectDependencies() []string {
	seen := make(map[string]bool)
	var deps []string

	for _, col := range e.Columns {
		parent, isFK := col.FK()
		// Ignore self-referencing foreign keys and duplicates
		if isFK && parent != e.Name && !seen[parent] {
			seen[parent] = true
			deps = append(deps, parent)
		}
	}

	return deps
}

// GenerationRule defines how a specific column generates fake data.
type GenerationRule struct {
	Type         string // "regex", "random_string", "random_int", "email", "foreign_key", "custom"
	Min          int
	Max          int
	RegexPattern string
}

type Bound struct {
	Lower int
	Upper int
}
