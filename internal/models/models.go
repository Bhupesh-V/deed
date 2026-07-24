package models

type Entity struct {
	Schema  string
	Name    string
	Columns []Column
	// Constraints []Constraint
}

type Column struct {
	Name string
	// VARCHAR, NUMERIC, etc
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
