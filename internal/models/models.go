package models

type Entity struct {
	Schema      string
	Name        string
	Columns     []Column
	Constraints []Constraint
}

type Column struct {
	Name string
	// VARCHAR, NUMERIC, etc
	Type DataType
}

// DataType captures the full PostgreSQL type signature.
type DataType struct {
	BaseType  string // e.g., "VARCHAR", "NUMERIC", "INT"
	Precision *int   // e.g., 10 for NUMERIC(10,2) or 255 for VARCHAR(255)
	Scale     *int   // e.g., 2 for NUMERIC(10,2)
}

type Constraint struct {
	Name string // Explicit constraint names are critical in Postgres
	Type ConstraintType

	// Fields populated based on the ConstraintType:
	Columns    []string           // Used by PK, Unique, FK, and Exclusion
	CheckExpr  string             // Used ONLY by Check (e.g., "price > 0")
	ForeignKey *ForeignKeyDetails // Used ONLY by ForeignKey
}

type ForeignKeyDetails struct {
	ParentTable   string
	ColumnMapping map[string]string // local_column -> parent_column
	OnDelete      string            // e.g., "CASCADE", "SET NULL"
	OnUpdate      string            // e.g., "RESTRICT"
}

